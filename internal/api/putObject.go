package api

import (
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/rstdm/glados/internal/api/middleware"
	"github.com/rstdm/glados/internal/api/object"
	"net/http"
)

func (a *API) putObject(c *gin.Context) {
	// TODO this function (putObject) is called before the request body (the file) has completely been transmitted.
	// TODO c.FormFile blocks until the file has completely been transmitted. -> Check weather the object already
	// TODO exists before calling c.FormFile. This way the request can be aborted without transmitting the file.
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

	objectHash := middleware.GetObjectHash(c)

	err = a.objectHandler.Write(objectHash, formFile)
	if err == nil {
		c.String(http.StatusOK, "object persisted")
		return
	}

	// there was an error
	if errors.Is(err, object.ErrObjectDoesExist) {
		c.String(http.StatusConflict, "The requested object already exists.")
	} else {
		err = fmt.Errorf("persist object: %w", err)
		_ = c.AbortWithError(http.StatusInternalServerError, err)
	}
}
