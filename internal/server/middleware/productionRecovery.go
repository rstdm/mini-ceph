package middleware

import (
	"fmt"
	"github.com/pkg/errors"
	"net"
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func ProductionRecovery(sugar *zap.SugaredLogger) gin.HandlerFunc {
	// this code is based on https://github.com/gin-gonic/gin/blob/master/recovery.go#L51
	return func(c *gin.Context) {
		defer func() {
			if rec := recover(); rec != nil {
				// Check for a broken connection, as it is not really a
				// condition that warrants a panic stack trace.
				if ne, ok := rec.(*net.OpError); ok {
					if se, ok := ne.Err.(*os.SyscallError); ok {
						if strings.Contains(strings.ToLower(se.Error()), "broken pipe") || strings.Contains(strings.ToLower(se.Error()), "connection reset by peer") {
							// If the connection is dead, we can't write a status to it.
							_ = c.Error(ne) // gin returns the processed error; it's no new exception
							c.Abort()
							return
						}
					}
				}

				if err, ok := rec.(error); ok {
					err = errors.WithStack(err)                               // add the stack trace to the line that caused the panic
					_ = c.AbortWithError(http.StatusInternalServerError, err) // gin returns the processed error; it's no new exception
				} else {
					sugar.Warnw("recovered with non error panic value", "value", rec, "type", fmt.Sprintf("%T", rec))
					c.AbortWithStatus(http.StatusInternalServerError)
				}

			}
		}()

		c.Next()
	}
}
