package api

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/rstdm/glados/internal/api/file"
	"go.uber.org/zap"
)

const objectParam = "objectHash"
const objectRoute = ":" + objectParam

type API struct {
	fileHandler        *file.Handler
	maxObjectSizeBytes int64
}

func NewAPI(objectFolder string, maxObjectSizeBytes int64, sugar *zap.SugaredLogger) (*API, error) {
	fileHandler, err := file.NewHandler(objectFolder, sugar)
	if err != nil {
		err = fmt.Errorf("create file handler: %w", err)
		return nil, err
	}

	api := &API{
		fileHandler:        fileHandler,
		maxObjectSizeBytes: maxObjectSizeBytes,
	}

	return api, err
}

func (a *API) RegisterHandler(engine *gin.Engine) {
	objectGroup := engine.Group(objectRoute, objectMiddleware)

	objectGroup.PUT("", a.putObject)
	objectGroup.GET("", a.getObject)
}
