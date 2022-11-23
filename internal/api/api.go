package api

import (
	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
	"github.com/rstdm/glados/internal/api/file"
	"go.uber.org/zap"
)

type API struct {
	fileHandler *file.Handler
}

func NewAPI(objectFolder string, sugar *zap.SugaredLogger) (*API, error) {
	fileHandler, err := file.NewHandler(objectFolder, sugar)
	if err != nil {
		err = errors.Wrap(err, "create file handler")
		return nil, err
	}

	api := &API{
		fileHandler: fileHandler,
	}

	return api, err
}

func (a *API) RegisterHandler(engine *gin.Engine) {
	engine.GET("/", a.helloWorldHandler)
}
