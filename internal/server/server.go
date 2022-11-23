package server

import (
	"context"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/rstdm/glados/internal/api"
	"github.com/rstdm/glados/internal/server/middleware"
	"go.uber.org/zap"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

type Server struct {
	router *gin.Engine
	server *http.Server
	sugar  *zap.SugaredLogger
}

func (s *Server) Launch() {
	go func() {
		if err := s.start(); err != nil {
			s.sugar.Fatalw("Encountered exception during server initialization; terminating", "error", err)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal)
	// kill (no param) default send syscall.SIGTERM
	// kill -2 is syscall.SIGINT
	// kill -9 is syscall.SIGKILL but can't be catch, so don't need add it
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	s.StopGraceful()
}

func (s *Server) start() error {
	s.sugar.Infow("Starting server", "address", s.server.Addr)
	err := s.server.ListenAndServe()
	if err == http.ErrServerClosed {
		return nil
	}
	return err
}

func (s *Server) StopGraceful() {
	s.sugar.Infow("Shutting down server gracefully")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := s.server.Shutdown(ctx); err != nil {
		s.sugar.Fatalw("Server forced to shutdown:", "error", err)
	}
}

func New(port int, bearerToken string, objectFolder string, maxObjectSizeBytes int64, useProductionLogger bool, sugar *zap.SugaredLogger) (*Server, error) {
	gin.SetMode(gin.ReleaseMode)

	router := gin.New()
	if err := router.SetTrustedProxies(nil); err != nil {
		return nil, fmt.Errorf("set trusted proxies to nil: %w", err)
	}

	middlewares := buildMiddlewares(bearerToken, useProductionLogger, sugar)
	router.Use(middlewares...)

	a, err := api.NewAPI(objectFolder, maxObjectSizeBytes, sugar)
	if err != nil {
		err = fmt.Errorf("create api: %w", err)
		return nil, err
	}
	a.RegisterHandler(router)

	httpServer := &http.Server{
		Addr:    fmt.Sprintf(":%v", port),
		Handler: router,
	}
	server := &Server{
		router: router,
		server: httpServer,
		sugar:  sugar,
	}

	return server, nil
}

func buildMiddlewares(bearerToken string, useProductionLogger bool, sugar *zap.SugaredLogger) []gin.HandlerFunc {
	var middlewares []gin.HandlerFunc

	if useProductionLogger {
		middlewares = append(middlewares, middleware.ProductionLogger(sugar), middleware.ProductionRecovery(sugar))
	} else {
		middlewares = append(middlewares, gin.Logger(), gin.Recovery())
	}

	if bearerToken == "" {
		sugar.Warn("No bearer token has been set and therefore all endpoints can be accessed without validation.")
	} else {
		middlewares = append(middlewares, middleware.BearerAuthentication(bearerToken))
	}

	return middlewares
}
