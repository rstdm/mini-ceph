package api

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/rstdm/mini-ceph/internal/api/middleware"
	"github.com/rstdm/mini-ceph/internal/api/object"
	"github.com/rstdm/mini-ceph/internal/api/object/distribution"
	"github.com/rstdm/mini-ceph/internal/configuration"
	"go.uber.org/zap"
)

const objectRoute = "object/:" + middleware.ObjectParam
const clusterRoute = "internal/:" + middleware.ObjectParam

type API struct {
	objectHandler       *object.Handler
	distributionHandler *distribution.Handler
	maxObjectSizeBytes  int64
	userBearerToken     string
	clusterBearerToken  string
	sugar               *zap.SugaredLogger
}

func NewAPI(config configuration.Configuration, sugar *zap.SugaredLogger) (*API, error) {
	distributionHandler := distribution.NewHandler(config.NodeID, config.NodeHosts, config.PlacementGroups)
	objectHandler, err := object.NewHandler(config.ObjectFolder, config.ClusterBearerToken, distributionHandler, sugar)
	if err != nil {
		err = fmt.Errorf("create object handler: %w", err)
		return nil, err
	}

	api := &API{
		objectHandler:       objectHandler,
		distributionHandler: distributionHandler,
		maxObjectSizeBytes:  config.MaxObjectSizeBytes,
		userBearerToken:     config.UserBearerToken,
		clusterBearerToken:  config.ClusterBearerToken,
		sugar:               sugar,
	}

	return api, err
}

func (a *API) RegisterHandler(engine *gin.Engine) {
	a.registerObjectRoutes(engine)
	a.registerClusterRoutes(engine)
}

func (a *API) registerObjectRoutes(engine *gin.Engine) {
	var middlewares []gin.HandlerFunc
	if a.userBearerToken != "" {
		middlewares = append(middlewares, middleware.BearerAuthentication(a.userBearerToken))
	} else {
		a.sugar.Warn("No userBearerToken has been specified. All user level API endpoints are exposed without authentication.")
	}
	middlewares = append(middlewares, middleware.ObjectMiddleware, middleware.DistributionMiddleware(false, a.distributionHandler))

	objectGroup := engine.Group(objectRoute, middlewares...)

	objectGroup.PUT("", a.putObject)
	objectGroup.GET("", a.getObject)
	objectGroup.DELETE("", a.deleteObject)
}

func (a *API) registerClusterRoutes(engine *gin.Engine) {
	var middlewares []gin.HandlerFunc
	if a.clusterBearerToken != "" {
		middlewares = append(middlewares, middleware.BearerAuthentication(a.clusterBearerToken))
	} else {
		a.sugar.Warn("No clusterBearerToken has been specified. All user level API endpoints are exposed without authentication.")
	}
	middlewares = append(middlewares, middleware.ObjectMiddleware, middleware.DistributionMiddleware(true, a.distributionHandler))

	clusterGroup := engine.Group(clusterRoute, middlewares...)

	clusterGroup.PUT("", a.putObject)
	clusterGroup.GET("", a.getObject)
	clusterGroup.DELETE("", a.deleteObject)
}
