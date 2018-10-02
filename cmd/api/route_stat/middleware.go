package route_stat

import (
	"flag"
	"sync"
	"time"

	"github.com/airbrake/gobrake"
	"github.com/gin-gonic/gin"
)

var notifier *gobrake.Notifier
var pathMap map[string]string
var lock = sync.RWMutex{}

var env = flag.String("env", "development", "environment, e.g. development or production")
var host = flag.String("host", "", "host")
var projectId = flag.Int64("project_id", 0, "project ID")
var projectKey = flag.String("project_key", "", "project key")

func init() {
	flag.Parse()

	notifier = gobrake.NewNotifierWithOptions(&gobrake.NotifierOptions{
		ProjectId:   *projectId,
		ProjectKey:  *projectKey,
		Host:        *host,
		Environment: *env,
	})
}

func NewAirbrakeMiddleware(engine *gin.Engine) func(c *gin.Context) {
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

	lock.RLock()
	defer lock.RUnlock()

	return pathMap[c.HandlerName()]
}

func extractRouteNames(engine *gin.Engine) {
	lock.RLock()
	if pathMap != nil {
		return
	}
	lock.RUnlock()

	lock.Lock()
	defer lock.Unlock()

	pathMap = make(map[string]string)
	for _, ri := range engine.Routes() {
		pathMap[ri.Handler] = ri.Path
	}
}
