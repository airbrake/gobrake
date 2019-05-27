package gin

import (
	"sync"

	"github.com/airbrake/gobrake"
	"github.com/gin-gonic/gin"
)

func NewMiddleware(engine *gin.Engine, notifier *gobrake.Notifier) func(c *gin.Context) {
	return func(c *gin.Context) {
		routeName := routeName(c, engine)
		_, trace := gobrake.NewRouteTrace(nil, c.Request.Method, routeName)

		c.Next()

		trace.StatusCode = c.Writer.Status()
		notifier.Routes.Notify(nil, trace)
	}
}

func routeName(c *gin.Context, engine *gin.Engine) string {
	initPathMap(engine)
	route, ok := pathMap[c.HandlerName()]
	if ok {
		return route
	}
	return "UNKNOWN"
}

var (
	pathMapOnce sync.Once
	pathMap     map[string]string
)

func initPathMap(engine *gin.Engine) {
	pathMapOnce.Do(func() {
		pathMap = make(map[string]string)
		for _, ri := range engine.Routes() {
			pathMap[ri.Handler] = ri.Path
		}
	})
}
