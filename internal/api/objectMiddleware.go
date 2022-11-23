package api

import (
	"github.com/gin-gonic/gin"
	"github.com/rstdm/glados/internal/api/object"
	"net/http"
)

const objectHashKey = "objectHash"

func objectMiddleware(c *gin.Context) {
	objectHash := c.Param(objectParam)
	if !object.IsObjectHash(objectHash) {
		c.String(http.StatusBadRequest, "Missing or invalid object hash")
		c.Abort()
		return
	}

	c.Set(objectHashKey, objectHash)

	c.Next()
}

func getObjectHash(c *gin.Context) string {
	return c.MustGet(objectHashKey).(string)
}
