package api

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"net/http"
)

func (a *API) putObject(c *gin.Context) {
	file, err := c.FormFile("file")
	if err != nil {
		c.String(http.StatusBadRequest, "Missing form-file 'file'")
		return
	}

	if file.Size > a.maxObjectSizeBytes {
		message := fmt.Sprintf("The object size is bigger than the configured threshold of %v bytes.", a.maxObjectSizeBytes)
		c.String(http.StatusRequestEntityTooLarge, message)
		return
	}
}
