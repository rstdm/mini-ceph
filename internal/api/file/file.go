package file

import (
	"errors"
	"fmt"
	"go.uber.org/zap"
	"io"
	"os"
)

func fileExists(path string) (exists bool, err error) {
	// https://stackoverflow.com/questions/12518876/how-to-check-if-a-file-exists-in-go
	_, err = os.Stat(path)
	if err == nil {
		return true, nil
	}

	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}

	err = fmt.Errorf("stat %v: %w", path, err)
	return false, err
}

func closeAndLogError(c io.Closer, resourceName string, sugar *zap.SugaredLogger) {
	if err := c.Close(); err != nil {
		sugar.Warnw("Failed to close resource",
			"err", err,
			"resourceType", fmt.Sprintf("%T", c),
			"resourceName", resourceName,
		)
	}
}
