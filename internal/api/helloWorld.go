package api

import (
	"github.com/gin-gonic/gin"
	"net/http"
)

func helloWorldHandler(context *gin.Context) {
	context.String(http.StatusOK, "Hello World")
}
