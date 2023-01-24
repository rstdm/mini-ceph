package api

import (
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/rstdm/mini-ceph/internal/api/middleware"
	"github.com/rstdm/mini-ceph/internal/api/object"
	"net/http"
)

func (a *API) deleteObject(c *gin.Context) {
	objectHash := middleware.GetObjectHash(c)

	err := a.objectHandler.Delete(objectHash)
	if err != nil && errors.Is(err, object.ErrObjectDoesNotExist) {
		c.String(http.StatusNotFound, "the requested object does not exists")
		return
	}
	if err != nil {
		err = fmt.Errorf("delete object: %w", err)
		_ = c.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	c.String(http.StatusOK, "object deleted")
}
