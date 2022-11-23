package file

import (
	"fmt"
	"go.uber.org/zap"
	"os"
	"path/filepath"
)

type Handler struct {
	objectFolder string
	sugar        *zap.SugaredLogger
}

func NewHandler(objectFolder string, sugar *zap.SugaredLogger) (*Handler, error) {
	absFolder, err := filepath.Abs(objectFolder)
	if err != nil {
		err = fmt.Errorf("convert [objectFolder=%v] to abs path: %w", objectFolder, err)
		return nil, err
	}

	if err := os.MkdirAll(absFolder, 0600); err != nil {
		err = fmt.Errorf("create [absFolder=%v]: %w", absFolder, err)
		return nil, err
	}

	handler := &Handler{
		objectFolder: absFolder,
		sugar:        sugar,
	}

	return handler, nil
}
