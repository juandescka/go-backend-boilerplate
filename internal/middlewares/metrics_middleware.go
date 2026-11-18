package middlewares

import (
	"strconv"
	"time"

	appmetrics "hris-core-service/internal/shared/metrics"

	"github.com/gin-gonic/gin"
)

// HTTPMetrics records one bounded observation after Gin has resolved the route and status.
func HTTPMetrics(recorder appmetrics.HTTPRecorder, excludedPath string) gin.HandlerFunc {
	if recorder == nil {
		recorder = appmetrics.NoopRecorder{}
	}
	return func(c *gin.Context) {
		if c.Request != nil && c.Request.URL != nil && c.Request.URL.Path == excludedPath {
			c.Next()
			return
		}

		startedAt := time.Now()
		recorder.RequestStarted()
		defer func() {
			route := c.FullPath()
			if route == "" {
				route = appmetrics.RouteUnmatched
			}
			recorder.RequestFinished(c.Request.Method, route, strconv.Itoa(c.Writer.Status()), time.Since(startedAt))
		}()
		c.Next()
	}
}
