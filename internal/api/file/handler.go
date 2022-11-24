package file

import (
	"errors"
	"fmt"
	"github.com/rstdm/glados/internal/api/object"
	"go.uber.org/multierr"
	"go.uber.org/zap"
	"io"
	"mime/multipart"
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

	handler := &Handler{
		objectFolder: absFolder,
		sugar:        sugar,
	}

	return handler, nil
}

func (h *Handler) PersistObject(objectHash string, file *multipart.FileHeader) error {
	objectPath, err := h.getObjectPath(objectHash)
	if err != nil {
		return fmt.Errorf("get object path: %w", err)
	}

	fileExists, err := h.fileExists(objectPath)
	if err != nil {
		return fmt.Errorf("check file existence: %w", err)
	}
	if fileExists {
		return ErrObjectAlreadyExists
	}

	// the object doesn't exit

	openedFile, err := file.Open()
	if err != nil {
		return fmt.Errorf("open multipart file header: %w", err)
	}
	defer h.closeAndLogError(openedFile, objectHash)

	if err := h.createObject(objectPath, openedFile); err == nil {
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
	objectPath, err := h.getObjectPath(objectHash)
	if err != nil {
		return "", fmt.Errorf("get object path: %w", err)
	}

	objectExists, err := h.fileExists(objectPath)
	if err != nil {
		return "", fmt.Errorf("file exists: %w", err)
	}

	if objectExists {
		return objectPath, nil
	} else {
		return "", nil
	}
}

func (h *Handler) fileExists(path string) (exists bool, err error) {
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

func (h *Handler) getObjectPath(objectHash string) (string, error) {
	// ensure that we're really dealing with an object hash before we're using it to create a path
	if !object.IsObjectHash(objectHash) {
		return "", errNoObjectHash
	}

	// this operation would be insecure if we hadn't validated the object hash before
	// For example an object hash like "../../someSystemConfigFile.xml" would be dangerous!
	objectPath := filepath.Join(h.objectFolder, objectHash)

	return objectPath, nil
}

func (h *Handler) createObject(objectPath string, file multipart.File) error {
	createdFile, err := os.Create(objectPath)
	if err != nil {
		return fmt.Errorf("create file: %w", err)
	}
	defer h.closeAndLogError(createdFile, objectPath)

	if _, err := io.Copy(createdFile, file); err != nil {
		return fmt.Errorf("write content to file: %w", err)
	}

	return nil
}

func (h *Handler) closeAndLogError(c io.Closer, resourceName string) {
	if err := c.Close(); err != nil {
		h.sugar.Warnw("Failed to close resource",
			"err", err,
			"resourceType", fmt.Sprintf("%T", c),
			"resourceName", resourceName,
		)
	}
}
