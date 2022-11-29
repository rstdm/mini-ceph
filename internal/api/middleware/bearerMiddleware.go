package middleware

import (
	"crypto/subtle"
	"github.com/gin-gonic/gin"
	"net/http"
	"strings"
)

func BearerAuthentication(bearerToken string) func(c *gin.Context) {
	byteBearer := []byte(bearerToken)

	return func(c *gin.Context) {
		providedAuthorization := c.Request.Header.Get("Authorization")
		providedBearer := strings.TrimPrefix(providedAuthorization, "Bearer ")

		if providedAuthorization == providedBearer {
			c.String(http.StatusUnauthorized, "Authorization header was empty or didn't contain a bearer token")
			c.Abort()
			return
		}
		if subtle.ConstantTimeCompare([]byte(providedBearer), byteBearer) == 0 {
			c.String(http.StatusUnauthorized, "The provided bearer token is invalid")
			c.Abort()
			return
		}

		c.Next()
	}
}
