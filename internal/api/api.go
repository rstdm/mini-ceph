package api

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/rstdm/glados/internal/api/middleware"
	"github.com/rstdm/glados/internal/api/object"
	"go.uber.org/zap"
)

const objectRoute = "object/:" + middleware.ObjectParam

type API struct {
	objectHandler      *object.Handler
	maxObjectSizeBytes int64
}

func NewAPI(objectFolder string, maxObjectSizeBytes int64, sugar *zap.SugaredLogger) (*API, error) {
	objectHandler, err := object.NewHandler(objectFolder, sugar)
	if err != nil {
		err = fmt.Errorf("create object handler: %w", err)
		return nil, err
	}

	api := &API{
		objectHandler:      objectHandler,
		maxObjectSizeBytes: maxObjectSizeBytes,
	}

	return api, err
}

func (a *API) RegisterHandler(engine *gin.Engine, userBearerToken string, sugar *zap.SugaredLogger) {
	var middlewares []gin.HandlerFunc
	if userBearerToken != "" {
		middlewares = append(middlewares, middleware.BearerAuthentication(userBearerToken))
	} else {
		sugar.Warn("No userBearerToken has been specified. All user level API endpoints are exposed without authentication.")
	}
	middlewares = append(middlewares, middleware.ObjectMiddleware)

	objectGroup := engine.Group(objectRoute, middlewares...)

	objectGroup.PUT("", a.putObject)
	objectGroup.GET("", a.getObject)
	objectGroup.DELETE("", a.deleteObject)
}
