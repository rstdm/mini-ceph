package api

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"net/http"
)

func (a *API) getObject(c *gin.Context) {
	objectHash := getObjectHash(c)

	objectPath, err := a.fileHandler.GetObjectPath(objectHash)
	if err != nil {
		err = fmt.Errorf("get object path: %w", err)
		_ = c.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	if objectPath == "" { // object doesn't exist
		c.String(http.StatusNotFound, "The requested object does not exist")
		return
	}

	c.File(objectPath)
	c.Status(http.StatusOK)
}
