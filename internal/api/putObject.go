package api

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"net/http"
)

func (a *API) putObject(c *gin.Context) {
	c.String(http.StatusOK, fmt.Sprintf("Put Object %v", getObjectHash(c)))
}
