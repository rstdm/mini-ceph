package middleware

import (
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func ProductionLogger(sugar *zap.SugaredLogger) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Start timer
		start := time.Now()

		// Process request
		c.Next()

		status := c.Writer.Status()
		isClientError := status >= 400 && status < 500
		isServerError := status >= 500 && status < 600

		// Stop timer
		latency := time.Now().Sub(start)

		path := c.Request.URL.Path
		raw := c.Request.URL.RawQuery
		if raw != "" {
			path = path + "?" + raw
		}

		errs := make([]error, len(c.Errors))
		for i, err := range c.Errors {
			errs[i] = err.Err
		}

		logArguments := []interface{}{"method", c.Request.Method, "path", path, "status", status, "errors", errs, "latency", latency}

		switch {
		case isServerError || len(c.Errors) > 0: // a non 500 status code doesn't mean that there were no errors
			sugar.Errorw("Server error", logArguments...)
		case isClientError:
			sugar.Debugw("Client error", logArguments...)
		default:
			sugar.Debugw("Request", logArguments...)
		}
	}
}
