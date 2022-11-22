package api

import (
	"github.com/gin-gonic/gin"
)

func RegisterHandler(engine *gin.Engine) {
	engine.GET("/", helloWorldHandler)
}
