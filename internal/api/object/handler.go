package object

import (
	"errors"
	"fmt"
	"github.com/rstdm/glados/internal/api/object/file"
	"github.com/rstdm/glados/internal/api/object/operationstate"
	"go.uber.org/zap"
	"mime/multipart"
)

var ErrObjectAlreadyExists = file.ErrObjectAlreadyExists

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

func (h *Handler) GetObjectPath(objectHash string) (string, error) {
	return h.fileHandler.GetObjectPath(objectHash) // TODO improve
}

func (h *Handler) PersistObject(objectHash string, formFile *multipart.FileHeader) error {
	return h.fileHandler.PersistObject(objectHash, formFile) // TODO improve
}
