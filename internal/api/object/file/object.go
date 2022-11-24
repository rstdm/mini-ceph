package file

import (
	"fmt"
	"github.com/rstdm/glados/internal/api/object/hash"
	"go.uber.org/zap"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
)

func getObjectPath(objectHash string, objectFolder string) (string, error) {
	// ensure that we're really dealing with an object hash before we're using it to create a path
	if !hash.IsObjectHash(objectHash) {
		return "", errNoObjectHash
	}

	// this operation would be insecure if we hadn't validated the object hash before
	// For example an object hash like "../../someSystemConfigFile.xml" would be dangerous!
	objectPath := filepath.Join(objectFolder, objectHash)

	return objectPath, nil
}

func createObject(objectPath string, file multipart.File, sugar *zap.SugaredLogger) error {
	createdFile, err := os.Create(objectPath)
	if err != nil {
		return fmt.Errorf("create file: %w", err)
	}
	defer closeAndLogError(createdFile, objectPath, sugar)

	if _, err := io.Copy(createdFile, file); err != nil {
		return fmt.Errorf("write content to file: %w", err)
	}

	return nil
}
