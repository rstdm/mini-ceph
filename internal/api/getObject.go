package api

import (
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/rstdm/glados/internal/api/middleware"
	"github.com/rstdm/glados/internal/api/object"
	"net/http"
)

func (a *API) getObject(c *gin.Context) {
	objectHash := middleware.GetObjectHash(c)

	err := a.objectHandler.TransferObject(objectHash, a.transferObjectCallback(c))
	if err == nil {
		// we don't have to do anything; the callback already completed the request
		return
	}

	// there was an error

	if errors.Is(err, object.ErrObjectDoesNotExist) {
		c.String(http.StatusNotFound, "The requested object does not exist")
		return
	}

	// it's an unexpected error
	err = fmt.Errorf("transfer object: %w", err)
	_ = c.AbortWithError(http.StatusInternalServerError, err)
}

func (a *API) transferObjectCallback(c *gin.Context) func(objectPath string) {
	return func(objectPath string) {
		c.File(objectPath) // This function also sets the response status to 200 OK
	}
}
