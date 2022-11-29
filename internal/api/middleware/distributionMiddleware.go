package middleware

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/rstdm/glados/internal/api/object/distribution"
	"net/http"
)

func DistributionMiddleware(isClusterEndpoint bool, distributionHandler *distribution.Handler) gin.HandlerFunc {
	return func(c *gin.Context) {
		objectHash := GetObjectHash(c)

		dist, err := distributionHandler.GetDistribution(objectHash)
		if err != nil {
			err = fmt.Errorf("calculate distribution: %w", err)
			_ = c.AbortWithError(http.StatusInternalServerError, err)
			return
		}

		nonPrimaryRequest := !dist.IsPrimary && !isClusterEndpoint // non-cluster endpoints must only be directed to the primary
		if nonPrimaryRequest || !dist.IsInPlacementGroup {
			message := fmt.Sprintf("Wrong node. This request should be directed to the primary of placement "+
				"group %v at %v", dist.CorrectPlacementGroup, dist.PrimaryHost)
			c.String(http.StatusMisdirectedRequest, message)
			c.Abort()
			return
		}

		c.Next()
	}
}
