package object

import (
	"errors"
	"fmt"
	"github.com/rstdm/glados/internal/api/object/distribution"
	"github.com/rstdm/glados/internal/api/object/file"
	"go.uber.org/zap"
	"mime/multipart"
	"sync"
)

type fsOperationResult string

const (
	fsOperationAllowed fsOperationResult = "allowed"
	fsOperationDenied  fsOperationResult = "denied"
	fsOperationError   fsOperationResult = "error"
)

var (
	ErrLookupError        = errors.New("error during object lookup")
	ErrObjectDoesNotExist = errors.New("the object does not exist")
	ErrObjectDoesExist    = errors.New("the object already exists")
)

type MutexEntry struct {
	wantRead                 []chan fsOperationResult
	read                     int
	wantWrite                chan fsOperationResult
	write                    bool
	wantDelete               chan fsOperationResult
	delete                   bool
	scheduledDelayedDeletion bool

	runningFSCheck bool
}

type Handler struct {
	mu        sync.Mutex
	mutexDict map[string]MutexEntry

	operationHandler *operationHandler
	fileHandler      *file.Handler
	sugar            *zap.SugaredLogger
}

func NewHandler(objectFolder string, clusterBearerToken string, distributionHandler *distribution.Handler, sugar *zap.SugaredLogger) (*Handler, error) {
	fileHandler, err := file.NewHandler(objectFolder, sugar)
	if err != nil {
		err = fmt.Errorf("create file handler: %w", err)
		return nil, err
	}

	operationHandler, err := newOperationHandler(clusterBearerToken, fileHandler, distributionHandler, sugar)
	if err != nil {
		err = fmt.Errorf("create newOperationHandler: %w", err)
		return nil, err
	}

	handler := &Handler{
		mutexDict:        map[string]MutexEntry{},
		operationHandler: operationHandler,
		fileHandler:      fileHandler,
		sugar:            sugar,
	}
	return handler, nil
}

func (f *Handler) Read(object string, transferObjectFunc func(objectPath string)) error {
	f.mu.Lock()
	entry := f.getEntry(object)

	if fileIsModified(entry) || entry.wantDelete != nil {
		f.mu.Unlock()
		return ErrObjectDoesNotExist
	}

	canReadImmediately := fileExistsWithoutLookup(entry)
	if canReadImmediately {
		entry.read += 1
		f.setEntry(object, entry)
		f.mu.Unlock()
	} else {
		setChannelFunc := func(entry *MutexEntry, c chan fsOperationResult) {
			entry.wantRead = append(entry.wantRead, c)
		}
		lookupResult := f.performFSLookup(object, entry, setChannelFunc) // this function unlocks the mutex
		switch lookupResult {
		case fsOperationError:
			return ErrLookupError
		case fsOperationDenied:
			return ErrObjectDoesNotExist
		}
	}

	// this performs the actual read; the error is returned at the end of the function
	readError := f.operationHandler.transferObject(object, transferObjectFunc)

	f.mu.Lock()
	entry = f.getEntry(object)
	entry.read -= 1
	f.setEntry(object, entry)
	f.mu.Unlock()

	if entry.read == 0 && entry.scheduledDelayedDeletion {
		if err := f.operationHandler.deleteObject(object); err != nil {
			f.sugar.Errorw("Failed to delete object after the last read operation ended",
				"err", err,
				"object", object)
		}

		f.mu.Lock()
		entry = f.getEntry(object)
		entry.scheduledDelayedDeletion = false
		f.setEntry(object, entry)
		f.mu.Unlock()
	}

	if readError != nil {
		readError = fmt.Errorf("transferObject: %w", readError)
		return readError
	}

	return nil
}

func (f *Handler) Write(object string, formFile *multipart.FileHeader) error {
	f.mu.Lock()
	entry := f.getEntry(object)

	if fileIsModified(entry) || entry.wantWrite != nil || fileExistsWithoutLookup(entry) {
		f.mu.Unlock()
		return ErrObjectDoesExist // objects are immutable, if fileIsModified is true the object is either currently created or it is currently deleted
	}

	setChannelFunc := func(entry *MutexEntry, c chan fsOperationResult) {
		entry.wantWrite = c
	}
	lookupResult := f.performFSLookup(object, entry, setChannelFunc) // this function unlocks the mutex
	switch lookupResult {
	case fsOperationError:
		return ErrLookupError
	case fsOperationDenied:
		return ErrObjectDoesExist
	}

	// this function saves the object to disk; the error is returned at the end of this function
	persistError := f.operationHandler.persistObject(object, formFile)

	f.mu.Lock()
	entry = f.getEntry(object)
	entry.write = false
	f.setEntry(object, entry)
	f.mu.Unlock()

	if persistError != nil {
		persistError = fmt.Errorf("persistObject: %w", persistError)
		return persistError
	}

	return nil
}

func (f *Handler) Delete(object string) error {
	f.mu.Lock()
	entry := f.getEntry(object)

	if fileIsModified(entry) || entry.wantDelete != nil || entry.scheduledDelayedDeletion {
		f.mu.Unlock()
		return ErrObjectDoesNotExist // objects are immutable; the object is either created, deleted, or is scheduled for deletion
	}

	canDeleteImmediately := entry.read > 0 && !entry.runningFSCheck
	if canDeleteImmediately {
		entry.delete = true
	} else {
		setChannelFunc := func(entry *MutexEntry, c chan fsOperationResult) {
			entry.wantDelete = c
		}
		lookupResult := f.performFSLookup(object, entry, setChannelFunc) // this function unlocks the mutex
		switch lookupResult {
		case fsOperationError:
			return ErrLookupError
		case fsOperationDenied:
			return ErrObjectDoesNotExist
		}

		f.mu.Lock() // the call to f.performFSLookup unlocked the mutex
		entry = f.getEntry(object)
	}

	if entry.read > 0 {
		entry.delete = false
		entry.scheduledDelayedDeletion = true
		f.setEntry(object, entry)
		f.mu.Unlock()
		return nil
	}

	f.setEntry(object, entry)
	f.mu.Unlock()

	// this function performs the actual deletion; the error is returned at the end of this function
	deleteError := f.operationHandler.deleteObject(object)

	f.mu.Lock()
	entry = f.getEntry(object)
	entry.delete = false
	f.setEntry(object, entry)
	f.mu.Unlock()

	if deleteError != nil {
		deleteError = fmt.Errorf("deleteObject: %w", deleteError)
		return deleteError
	}

	return nil
}

func fileIsModified(entry MutexEntry) bool {
	return entry.write || entry.delete || entry.scheduledDelayedDeletion
}

func fileExistsWithoutLookup(entry MutexEntry) bool {
	return entry.read > 0 && entry.wantDelete == nil && !entry.scheduledDelayedDeletion
}

// performFSLookup requires that the mutex has already been locked. At the end of the function the mutex will be unlocked.
func (f *Handler) performFSLookup(object string, entry MutexEntry, setChannelFunc func(entry *MutexEntry, c chan fsOperationResult)) fsOperationResult {
	var c chan fsOperationResult

	if entry.runningFSCheck {
		c = make(chan fsOperationResult)
		setChannelFunc(&entry, c)

		f.setEntry(object, entry)
		f.mu.Unlock()
	} else {
		c = make(chan fsOperationResult, 1) // checkFS will send to this channel; if the channel wasn't buffered it would create a deadlock
		setChannelFunc(&entry, c)

		entry.runningFSCheck = true
		f.setEntry(object, entry)
		f.mu.Unlock()

		f.checkFS(object)
	}

	operationResult := <-c
	return operationResult
}

func (f *Handler) checkFS(object string) {
	objectExists, err := f.fileHandler.ObjectExists(object)
	if err != nil {
		f.sugar.Errorw("Could not check weather the object exists",
			"err", err,
			"object", object)
	}

	f.mu.Lock()
	defer f.mu.Unlock()
	entry := f.getEntry(object)
	entry.runningFSCheck = false

	if len(entry.wantRead) > 0 {
		var canReadResult fsOperationResult
		switch {
		case err != nil:
			canReadResult = fsOperationError
		case objectExists:
			canReadResult = fsOperationAllowed
			entry.read += len(entry.wantRead)
		default:
			canReadResult = fsOperationDenied
		}

		for _, canReadChannel := range entry.wantRead {
			if canReadChannel != nil {
				canReadChannel <- canReadResult
			}
		}
	}
	entry.wantRead = nil

	if entry.wantWrite != nil {
		switch {
		case err != nil:
			entry.wantWrite <- fsOperationError
		case objectExists:
			entry.wantWrite <- fsOperationDenied
		default:
			entry.write = true
			entry.wantWrite <- fsOperationAllowed
		}
	}
	entry.wantWrite = nil

	if entry.wantDelete != nil {
		switch {
		case err != nil:
			entry.wantDelete <- fsOperationError
		case objectExists:
			entry.delete = true
			entry.wantDelete <- fsOperationAllowed
		default:
			entry.wantDelete <- fsOperationDenied
		}
	}
	entry.wantDelete = nil

	f.mutexDict[object] = entry
}

func (f *Handler) getEntry(object string) MutexEntry {
	entry, ok := f.mutexDict[object]
	if ok {
		return entry
	}

	return MutexEntry{}
}

func (f *Handler) setEntry(object string, entry MutexEntry) {
	isEmpty := len(entry.wantRead) == 0 &&
		entry.read == 0 &&
		entry.wantWrite == nil &&
		!entry.write &&
		entry.wantDelete == nil &&
		!entry.delete &&
		!entry.scheduledDelayedDeletion &&
		!entry.runningFSCheck

	if isEmpty { // unfortunately the test entry == MutexEntry{} is not allowed
		delete(f.mutexDict, object)
	} else {
		f.mutexDict[object] = entry
	}
}
