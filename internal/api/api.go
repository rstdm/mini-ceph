package api

import (
	"github.com/gin-gonic/gin"
)

type API struct{}

func NewAPI() *API {
	return &API{}
}

func (a *API) RegisterHandler(engine *gin.Engine) {
	engine.GET("/", a.helloWorldHandler)
}
