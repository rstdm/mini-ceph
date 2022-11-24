package object

import (
	"fmt"
	"github.com/rstdm/glados/internal/api/object/file"
	"go.uber.org/zap"
)

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
