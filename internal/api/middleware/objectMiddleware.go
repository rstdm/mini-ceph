package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/rstdm/glados/internal/api"
	"github.com/rstdm/glados/internal/api/object/hash"
	"net/http"
)

const objectHashKey = "objectHash"

func objectMiddleware(c *gin.Context) {
	objectHash := c.Param(api.objectParam)
	if !hash.IsObjectHash(objectHash) {
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
