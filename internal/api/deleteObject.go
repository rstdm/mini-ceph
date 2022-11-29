package api

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/rstdm/glados/internal/api/middleware"
	"net/http"
)

func (a *API) deleteObject(c *gin.Context) {
	objectHash := middleware.GetObjectHash(c)

	didExist, err := a.objectHandler.DeleteObject(objectHash)
	if err != nil {
		err = fmt.Errorf("delete object: %w", err)
		_ = c.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	if didExist {
		c.String(http.StatusOK, "object deleted")
		return
	} else {
		c.String(http.StatusNotFound, "the requested object does not exists")
	}
}
