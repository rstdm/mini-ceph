package api

import (
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/rstdm/glados/internal/api/file"
	"net/http"
)

func (a *API) putObject(c *gin.Context) {
	formFile, err := c.FormFile("file")
	if err != nil {
		c.String(http.StatusBadRequest, "Missing form-file 'file'")
		return
	}

	if formFile.Size > a.maxObjectSizeBytes {
		message := fmt.Sprintf("The object size is bigger than the configured threshold of %v bytes.", a.maxObjectSizeBytes)
		c.String(http.StatusRequestEntityTooLarge, message)
		return
	}

	objectHash := getObjectHash(c)

	err = a.fileHandler.PersistObject(objectHash, formFile)
	if err == nil {
		c.String(http.StatusOK, "object persisted")
		return
	}

	// there was an error
	if errors.Is(err, file.ErrObjectAlreadyExists) {
		c.String(http.StatusConflict, "The requested object already exists.")
	} else {
		err = fmt.Errorf("persist object: %w", err)
		_ = c.AbortWithError(http.StatusInternalServerError, err)
	}
}
