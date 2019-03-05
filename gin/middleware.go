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
		end := time.Now()

		routeName := getRouteName(c, engine)
		notifier.Routes.Notify(nil, &gobrake.RouteTrace{
			Method:     c.Request.Method,
			Route:      routeName,
			StatusCode: c.Writer.Status(),
			Start:      start,
			End:        end,
		})
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
