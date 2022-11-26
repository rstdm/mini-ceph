package object

import (
	"errors"
	"fmt"
	"github.com/rstdm/glados/internal/api/object/file"
	"github.com/rstdm/glados/internal/api/object/operationstate"
	"go.uber.org/zap"
	"mime/multipart"
)

var (
	ErrObjectAlreadyExists = file.ErrObjectAlreadyExists
	ErrObjectDoesNotExist  = errors.New("object does not exist")
)

type Handler struct {
	fileHandler           *file.Handler
	operationStateHandler *operationstate.Handler
	sugar                 *zap.SugaredLogger
}

func NewHandler(objectFolder string, sugar *zap.SugaredLogger) (*Handler, error) {
	fileHandler, err := file.NewHandler(objectFolder, sugar)
	if err != nil {
		err = fmt.Errorf("create file handler: %w", err)
		return nil, err
	}

	operationStateHandler := operationstate.NewHandler(sugar)

	objectHandler := &Handler{
		fileHandler:           fileHandler,
		operationStateHandler: operationStateHandler,
		sugar:                 sugar,
	}

	return objectHandler, nil
}

func (h *Handler) DeleteObject(objectHash string) (didExist bool, err error) {
	if err := h.operationStateHandler.WantDelete(objectHash); err != nil {
		if errors.Is(err, operationstate.ErrOperationNotAllowed) {
			// the object is currently getting deleted by another coroutine
			return false, nil
		}

		err = fmt.Errorf("mark wantDelete operation as running: %w", err)
		return false, err
	}

	objectExists, err := h.fileHandler.ObjectExists(objectHash)
	if err != nil {
		h.operationStateHandler.AbortWantDelete(objectHash)
		err = fmt.Errorf("check object existence: %w", err)
		return false, err
	}
	if !objectExists {
		h.operationStateHandler.AbortWantDelete(objectHash)
		return false, nil
	}

	mustDelay, err := h.operationStateHandler.StartDeleting(objectHash, h.deleteCallback(objectHash))

	if err != nil {
		if errors.Is(err, operationstate.ErrOperationNotAllowed) {
			// pretend that the object doesn't exist
			// the object is either in the process of being created or it has already been marked for deletion
			return false, nil
		}

		// unexpected error
		err = fmt.Errorf("mark delete operation as running: %w", err)
		return false, err
	}
	if mustDelay {
		// pretend that the object has been deleted
		// to the outside world it looks as if the object is gone (e.g. it can no longer be read)
		if err := h.fileHandler.RemovePersistedFlag(objectHash); err != nil {
			err = fmt.Errorf("remove persisted flag: %w", err)
			return false, err
		}

		return true, nil
	}

	// DoneDeleting must not be called here if mustDelay is true. It will be called by the deleteCallback
	defer h.operationStateHandler.DoneDeleting(objectHash)

	return h.fileHandler.DeleteObject(objectHash)
}

func (h *Handler) deleteCallback(objectHash string) func() {
	return func() {
		defer h.operationStateHandler.DoneDeleting(objectHash)

		if _, err := h.fileHandler.DeleteObject(objectHash); err != nil {
			h.sugar.Errorw("Failed to delete object",
				"err", err,
				"objectHash", objectHash,
			)
		}
	}
}

func (h *Handler) TransferObject(objectHash string, transferObjectFunc func(objectPath string)) error {
	objectExists, err := h.fileHandler.ObjectExists(objectHash)
	if err != nil {
		return fmt.Errorf("check object existence")
	}
	if !objectExists {
		return ErrObjectDoesNotExist
	}

	if err := h.operationStateHandler.StartReading(objectHash); err != nil {
		if errors.Is(err, operationstate.ErrOperationNotAllowed) {
			// the object is in the process of being created, or it has been marked for deletion
			return ErrObjectDoesNotExist
		}

		// unexpected error
		return fmt.Errorf("mark object as being read: %w", err)
	}
	defer h.operationStateHandler.DoneReading(objectHash)

	objectPath, err := h.fileHandler.GetObjectPath(objectHash)
	if err != nil {
		return fmt.Errorf("get object path: %w", err)
	}

	// we have to check again. Maybe the object was deleted after the last check and before the call to
	// operationStateHandler.StartReading
	if objectPath == "" {
		return ErrObjectDoesNotExist
	}

	transferObjectFunc(objectPath)

	return nil
}

func (h *Handler) PersistObject(objectHash string, formFile *multipart.FileHeader) error {
	if err := h.operationStateHandler.WantCreate(objectHash); err != nil {
		if errors.Is(err, operationstate.ErrOperationNotAllowed) {
			return ErrObjectAlreadyExists // another create operation is currently running
		}

		err = fmt.Errorf("mark wantCreate operation as running: %w", err)
		return err
	}

	objectExists, err := h.fileHandler.ObjectExists(objectHash)
	if err != nil {
		h.operationStateHandler.AbortWantCreate(objectHash)
		return fmt.Errorf("check object existence: %w", err)
	}
	if objectExists {
		h.operationStateHandler.AbortWantCreate(objectHash)
		return ErrObjectAlreadyExists
	}

	// we have to check that the object doesn't exist before we mark the create operation as ongoing
	// Situation 1: The object already exists -> The previous check for objectExists ensures that the create operation
	//				is not started.
	// Situation 2: The object doesn't exist -> The create operation is started. The operationStateHandler ensures that
	//				new operations on this object are denied until the create operation is done.
	if err := h.operationStateHandler.StartCreating(objectHash); err != nil {
		if errors.Is(err, operationstate.ErrOperationNotAllowed) {
			// The object is either in the process of being created, it is read (which means that it exists) or it has
			// already been marked for deletion.
			return ErrObjectAlreadyExists
		}

		// some unexpected error
		return fmt.Errorf("mark create operation as running: %w", err)
	}
	defer h.operationStateHandler.DoneCreating(objectHash)

	return h.fileHandler.PersistObject(objectHash, formFile)
}
