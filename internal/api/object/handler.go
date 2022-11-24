package object

import (
	"fmt"
	"github.com/rstdm/glados/internal/api/object/file"
	"go.uber.org/zap"
	"mime/multipart"
)

var ErrObjectAlreadyExists = file.ErrObjectAlreadyExists

type Handler struct {
	fileHandler *file.Handler
	sugar       *zap.SugaredLogger
}

func NewHandler(objectFolder string, sugar *zap.SugaredLogger) (*Handler, error) {
	fileHandler, err := file.NewHandler(objectFolder, sugar)
	if err != nil {
		err = fmt.Errorf("create file handler: %w", err)
		return nil, err
	}

	objectHandler := &Handler{
		fileHandler: fileHandler,
		sugar:       sugar,
	}

	return objectHandler, nil
}

func (h *Handler) DeleteObject(objectHash string) (didExist bool, err error) {
	return h.fileHandler.DeleteObject(objectHash) // TODO improve
}

func (h *Handler) GetObjectPath(objectHash string) (string, error) {
	return h.fileHandler.GetObjectPath(objectHash) // TODO improve
}

func (h *Handler) PersistObject(objectHash string, formFile *multipart.FileHeader) error {
	return h.fileHandler.PersistObject(objectHash, formFile) // TODO improve
}
