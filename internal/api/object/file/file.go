package file

import (
	"errors"
	"fmt"
	"go.uber.org/zap"
	"io"
	"io/fs"
	"os"
)

// persistedFileMode will be set for all files that should be kept. 0400 is read access for the current user and
// no access for anyone else. See https://wiki.ubuntuusers.de/chmod/ for details.
const persistedFileMode fs.FileMode = 0400

// notPersistedFileMode CAN (!) be set on a file to signal that it is not persisted. Note, however, that os.Create
// creates files with mode 0666. Use persistedFileMode to check weather a file is persisted.
// 0600 means read and write access for the current user and no access for anyone else. See
// https://wiki.ubuntuusers.de/chmod/ for details.
const notPersistedFileMode fs.FileMode = 0600

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

func markAsPersisted(file *os.File) error {
	if err := file.Chmod(persistedFileMode); err != nil {
		return fmt.Errorf("chmod %v: %w", persistedFileMode, err)
	}

	return nil
}

func isMarkedAsPersisted(fileInfo fs.FileInfo) bool {
	return fileInfo.Mode() == persistedFileMode
}

func removePersistedFlag(objectPath string) error {
	if err := os.Chmod(objectPath, notPersistedFileMode); err != nil {
		return fmt.Errorf("chmod %v: %w", notPersistedFileMode, err)
	}

	return nil
}
