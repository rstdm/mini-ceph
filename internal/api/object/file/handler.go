package file

import (
	"errors"
	"fmt"
	"go.uber.org/multierr"
	"go.uber.org/zap"
	"os"
	"path/filepath"
)

var (
	ErrObjectAlreadyExists = errors.New("object already exists")
	errNoObjectHash        = errors.New("provided object hash is no object hash")
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

	// 0700 = rwx (user) and no permissions for the group of the folder and for every other user
	// see https://wiki.ubuntuusers.de/chmod/ for details
	if err := os.MkdirAll(absFolder, 0700); err != nil {
		err = fmt.Errorf("create [absFolder=%v]: %w", absFolder, err)
		return nil, err
	}

	if err := purgeObjects(objectFolder, sugar); err != nil {
		err = fmt.Errorf("purge not persisted objects: %w", err)
		return nil, err
	}

	handler := &Handler{
		objectFolder: absFolder,
		sugar:        sugar,
	}

	return handler, nil
}

func (h *Handler) ObjectExists(objectHash string) (exists bool, err error) {
	objectPath, err := getObjectPath(objectHash, h.objectFolder)
	if err != nil {
		err = fmt.Errorf("get object path: %w", err)
		return false, err
	}

	exists, err = fileExists(objectPath)
	if err != nil {
		err = fmt.Errorf("check file existence: %w", err)
		return false, err
	}

	return exists, nil
}

func (h *Handler) PersistObject(objectHash string, objectContent []byte) error {
	objectPath, err := getObjectPath(objectHash, h.objectFolder)
	if err != nil {
		return fmt.Errorf("get object path: %w", err)
	}

	fileExists, err := fileExists(objectPath)
	if err != nil {
		return fmt.Errorf("check file existence: %w", err)
	}
	if fileExists {
		return ErrObjectAlreadyExists
	}

	// the object doesn't exit

	if err := createObject(objectPath, objectContent, h.sugar); err == nil {
		return nil
	}

	// the object couldn't be created
	err = fmt.Errorf("create object: %w", err)

	// the object couldn't be persisted in a file and the file might be in an inconsistent state. Remove the file
	// (if it exists) to ensure a consistent state.
	if removeErr := os.Remove(objectPath); removeErr != nil && !errors.Is(removeErr, os.ErrNotExist) {
		removeErr = fmt.Errorf("remove object file to ensure a consistent state: %w", removeErr)
		err = multierr.Append(err, removeErr)
	}

	return err
}

func (h *Handler) GetObjectPath(objectHash string) (string, error) {
	objectPath, err := getObjectPath(objectHash, h.objectFolder)
	if err != nil {
		return "", fmt.Errorf("get object path: %w", err)
	}

	objectExists, err := fileExists(objectPath)
	if err != nil {
		return "", fmt.Errorf("file exists: %w", err)
	}

	if objectExists {
		return objectPath, nil
	} else {
		return "", nil
	}
}

func (h *Handler) DeleteObject(objectHash string) error {
	objectPath, err := getObjectPath(objectHash, h.objectFolder)
	if err != nil {
		return fmt.Errorf("get object path: %w", err)
	}

	err = os.Remove(objectPath)
	if err != nil {
		err = fmt.Errorf("remove object: %w", err)
		return err
	}

	return nil
}

func (h *Handler) RemovePersistedFlag(objectHash string) error {
	objectPath, err := getObjectPath(objectHash, h.objectFolder)
	if err != nil {
		return fmt.Errorf("get object path: %w", err)
	}

	if err := removePersistedFlag(objectPath); err != nil {
		return fmt.Errorf("remove persisted flag: %w", err)
	}

	return nil
}

func purgeObjects(objectFolder string, sugar *zap.SugaredLogger) error {
	dirEntries, err := os.ReadDir(objectFolder)
	if err != nil {
		return fmt.Errorf("list files in object dir: %w", err)
	}

	var merr error
	for _, entry := range dirEntries {
		if entry.IsDir() {
			continue
		}

		info, err := entry.Info()
		if err != nil {
			err = fmt.Errorf("read file info of object %v: %w", entry.Name(), err)
			merr = multierr.Append(merr, err)
			continue
		}

		if isMarkedAsPersisted(info) {
			continue
		}

		path := filepath.Join(objectFolder, entry.Name())
		sugar.Infow("Deleting object that is not marked as persisted.",
			"object", entry.Name(),
			"path", path,
		)

		if err := os.Remove(path); err != nil {
			err = fmt.Errorf("remove %v: %w", path, err)
			merr = multierr.Append(merr, err)
		}
	}

	if merr != nil {
		return fmt.Errorf("process files: %w", err)
	}

	return nil
}
