package gin

import (
	"sync"
	"time"

	"github.com/airbrake/gobrake"
	"github.com/gin-gonic/gin"
)

var pathMapOnce sync.Once
var pathMap map[string]string

func NewMiddleware(engine *gin.Engine, notifier *gobrake.Notifier) func(c *gin.Context) {
	return func(c *gin.Context) {
		start := time.Now()

		c.Next()

		dur := time.Since(start)

		routeName := getRouteName(c, engine)
		notifier.IncRequest(c.Request.Method, routeName, c.Writer.Status(), dur, start)
	}
}

func getRouteName(c *gin.Context, engine *gin.Engine) string {
	extractRouteNames(engine)
	route, ok := pathMap[c.HandlerName()]
	if ok {
		return route
	}
	return "UNKNOWN"
}

func extractRouteNames(engine *gin.Engine) {
	pathMapOnce.Do(func() {
		pathMap = make(map[string]string)
		for _, ri := range engine.Routes() {
			pathMap[ri.Handler] = ri.Path
		}
	})
}
