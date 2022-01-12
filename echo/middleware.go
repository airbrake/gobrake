package echo

import (
	"log"
	"net/http"

	"github.com/airbrake/gobrake/v5"
	echo "github.com/labstack/echo/v4"
)

// NewMiddleware implements a middleware that can be used in echo
func NewMiddleware(n *gobrake.Notifier) echo.HandlerFunc {
	if n == nil {
		return echo.HandlerFunc(func(c echo.Context) (err error) {
			var next echo.HandlerFunc
			next(c)
			return
		})
	}
	return echo.HandlerFunc(func(c echo.Context) (err error) {
		var next echo.HandlerFunc
		route := echo.GetPath(c.Request())
		ctx := c.Request().Context()
		method := c.Request().Method
		ctx, routeMetric := gobrake.NewRouteMetric(ctx, method, route)
		arw := newAirbrakeEchoContext(c)
		next(arw)
		routeMetric.StatusCode = arw.statusCode
		err = n.Routes.Notify(ctx, routeMetric)
		if err != nil {
			log.Println("[airbrake/error]: ", err)
		}
		return
	})
}

type airbrakeEchoContext struct {
	echo.Context
	statusCode int
}

func newAirbrakeEchoContext(c echo.Context) *airbrakeEchoContext {
	return &airbrakeEchoContext{c, http.StatusOK}
}

func (arw *airbrakeEchoContext) WriteHeader(code int) {
	arw.statusCode = code
	arw.Echo().AcquireContext().Response().WriteHeader(code)
}
