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
	err = h.operationStateHandler.StartDeleting(objectHash, h.deleteCallback(objectHash))

	if err != nil && errors.Is(err, operationstate.ErrDeletionMustBeDelayed) {
		// pretend that the object has been deleted
		// to the outside world it looks as if the object is gone (e.g. it can no longer be read)
		return true, nil
	}

	// this defer must not be called if the deletion has been delayed
	defer h.operationStateHandler.DoneDeleting(objectHash)

	if err == nil {
		return h.fileHandler.DeleteObject(objectHash)
	}

	// there was an error

	if errors.Is(err, operationstate.ErrOperationNotAllowed) {
		// pretend that the object doesn't exist
		// the object is either in the process of being created or it has already been marked for deletion
		return false, nil
	}

	//unexpected error
	err = fmt.Errorf("mark object state as deleting: %w", err)
	return false, err
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
	objectExists, err := h.fileHandler.ObjectExists(objectHash)
	if err != nil {
		return fmt.Errorf("check object existence: %w", err)
	}
	if objectExists {
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
