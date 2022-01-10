package gin

import (
	"context"

	"github.com/airbrake/gobrake/v5"

	"github.com/gin-gonic/gin"
)

// New returns a function that satisfies gin.HandlerFunc interface
func New(notifier *gobrake.Notifier) gin.HandlerFunc {
	return func(c *gin.Context) {
		_, metric := gobrake.NewRouteMetric(context.TODO(), c.Request.Method, c.FullPath())

		c.Next()

		metric.StatusCode = c.Writer.Status()
		_ = notifier.Routes.Notify(context.TODO(), metric)
	}
}

// This function is deprecated. Please use New() function instead
func NewMiddleware(engine *gin.Engine, notifier *gobrake.Notifier) func(c *gin.Context) {
	return New(notifier)
}
