package replication

import (
	"bytes"
	"fmt"
	"github.com/go-resty/resty/v2"
	"go.uber.org/multierr"
	"go.uber.org/zap"
	"net/http"
	"path"
)

type Handler struct {
	client *resty.Client
	sugar  *zap.SugaredLogger
}

func NewHandler(clusterBearerToken string, sugar *zap.SugaredLogger) *Handler {
	client := resty.New()

	if clusterBearerToken != "" {
		client.SetAuthToken(clusterBearerToken)
	}

	return &Handler{
		client: client,
		sugar:  sugar,
	}
}

func (h *Handler) Replicate(objectHash string, objectContent []byte, hosts []string) error {
	var merr error

	for _, host := range hosts {
		if err := h.replicateToHost(objectHash, objectContent, host); err != nil {
			merr = fmt.Errorf("replicate to %v: %w", host, err)
			break
		}
	}

	if merr != nil {
		h.sugar.Errorw(
			"Failed to replicate object. Deleting created replicas from all hosts.",
			"err", merr,
			"objectHash", objectHash,
		)
		if err := h.Delete(objectHash, hosts); err != nil {
			err = fmt.Errorf("delete replicas after failed replication attempt: %w", err)
			merr = multierr.Append(merr, err)
		}

		return merr
	}

	return nil
}

func (h *Handler) replicateToHost(objectHash string, objectContent []byte, host string) error {
	url := buildURL(objectHash, host)
	reader := bytes.NewReader(objectContent)

	response, err := h.client.R().
		SetFileReader("file", "", reader).
		Put(url)
	if err != nil {
		return fmt.Errorf("PUT %v: %w", url, err)
	}

	if response.StatusCode() != http.StatusOK {
		return fmt.Errorf("PUT %v yielded unexpected http status code %v", url, response.StatusCode())
	}

	return nil
}

func (h *Handler) Delete(objectHash string, hosts []string) error {
	var merr error

	for _, host := range hosts {
		if err := h.deleteFromHost(objectHash, host); err != nil {
			err = fmt.Errorf("delete replica from host %v: %w", host, err)
			merr = multierr.Append(merr, err)
		}
	}

	return merr
}

func (h *Handler) deleteFromHost(objectHash string, host string) error {
	url := buildURL(objectHash, host)
	response, err := h.client.R().Delete(url)
	if err != nil {
		return fmt.Errorf("perform DELETE request to url %v: %w", url, err)
	}

	if response.StatusCode() != http.StatusOK {
		return fmt.Errorf("requested DELETE %v, server responded with unexpected status code %v", url, response.StatusCode())
	}

	return nil
}

func buildURL(objectHash string, host string) string {
	return "http://" + path.Join(host, "internal", objectHash)
}
