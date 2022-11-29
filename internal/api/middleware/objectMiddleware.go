package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/rstdm/glados/internal/api/object/hash"
	"net/http"
)

const ObjectParam = "objectHash"
const objectHashKey = "objectHash"

func ObjectMiddleware(c *gin.Context) {
	objectHash := c.Param(ObjectParam)
	if !hash.IsObjectHash(objectHash) {
		c.String(http.StatusBadRequest, "Missing or invalid object hash")
		c.Abort()
		return
	}

	c.Set(objectHashKey, objectHash)

	c.Next()
}

func GetObjectHash(c *gin.Context) string {
	return c.MustGet(objectHashKey).(string)
}
