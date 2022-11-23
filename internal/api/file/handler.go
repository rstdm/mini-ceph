package file

import (
	"github.com/pkg/errors"
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
		err = errors.Wrapf(err, "convert [objectFolder=%v] to abs path", objectFolder)
		return nil, err
	}

	if err := os.MkdirAll(absFolder, 0600); err != nil {
		err = errors.Wrapf(err, "create [absFolder=%v]", absFolder)
		return nil, err
	}

	handler := &Handler{
		objectFolder: absFolder,
		sugar:        sugar,
	}

	return handler, nil
}
