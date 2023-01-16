package object

import (
	"fmt"
	"github.com/rstdm/glados/internal/api/object/distribution"
	"github.com/rstdm/glados/internal/api/object/file"
	"github.com/rstdm/glados/internal/api/object/replication"
	"go.uber.org/multierr"
	"go.uber.org/zap"
	"io"
	"mime/multipart"
)

type operationHandler struct {
	distributionHandler *distribution.Handler
	replicationHandler  *replication.Handler
	fileHandler         *file.Handler
	sugar               *zap.SugaredLogger
}

func newOperationHandler(clusterBearerToken string, fileHandler *file.Handler, distributionHandler *distribution.Handler, sugar *zap.SugaredLogger) (*operationHandler, error) {
	replicationHandler := replication.NewHandler(clusterBearerToken, sugar)

	operationHandler := &operationHandler{
		distributionHandler: distributionHandler,
		replicationHandler:  replicationHandler,
		fileHandler:         fileHandler,
		sugar:               sugar,
	}

	return operationHandler, nil
}

func (h *operationHandler) deleteObject(objectHash string) error {

	dist, err := h.distributionHandler.GetDistribution(objectHash)
	if err != nil {
		err = fmt.Errorf("calculate distribution: %w", err)
		return err
	}

	if dist.IsPrimary {
		if err := h.replicationHandler.Delete(objectHash, dist.SlaveHosts); err != nil {
			h.sugar.Errorw("Failed to delete replicated copies of object", "err", err, "objectHash", objectHash)
		}
	}

	err = h.fileHandler.DeleteObject(objectHash)
	if err != nil {
		err = fmt.Errorf("delete local replica: %w", err)
		return err
	}

	return nil
}

func (h *operationHandler) transferObject(objectHash string, transferObjectFunc func(objectPath string)) error {

	objectPath, err := h.fileHandler.GetObjectPath(objectHash)
	if err != nil {
		return fmt.Errorf("get object path: %w", err)
	}

	// we have to check again. Maybe the object was deleted after the last check and before the call to
	// operationStateHandler.StartReading
	if objectPath == "" {
		return ErrObjectDoesNotExist
	}

	transferObjectFunc(objectPath)

	return nil
}

func (h *operationHandler) persistObject(objectHash string, formFile *multipart.FileHeader) error {

	openedFile, err := formFile.Open()
	if err != nil {
		return fmt.Errorf("open form file: %w", err)
	}
	defer file.CloseAndLogError(openedFile, "formFile", h.sugar)

	objectContent, err := io.ReadAll(openedFile)
	if err != nil {
		return fmt.Errorf("read formFile into memory: %w", err)
	}

	dist, err := h.distributionHandler.GetDistribution(objectHash)
	if err != nil {
		err = fmt.Errorf("calculate distribution: %w", err)
		return err
	}

	if dist.IsPrimary {
		if err := h.replicationHandler.Replicate(objectHash, objectContent, dist.SlaveHosts); err != nil {
			// we don't have to delete the objects because Replicate() deletes all already created objects if it
			// encounters an error.
			return fmt.Errorf("replicate object to slaves: %w", err)
		}
	}

	if err := h.fileHandler.PersistObject(objectHash, objectContent); err != nil {
		merr := fmt.Errorf("persist object locally: %w", err)
		if err = h.replicationHandler.Delete(objectHash, dist.SlaveHosts); err != nil {
			err = fmt.Errorf("delete replicated object because object could not be persisted locally: %w", err)
			merr = multierr.Append(merr, err)
		}
		return merr
	}

	return nil
}
