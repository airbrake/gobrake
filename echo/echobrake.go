package echo

import (
	"context"
	"log"

	"github.com/airbrake/gobrake/v5"
	"github.com/labstack/echo/v4"
)

type handler struct {
	notifier *gobrake.Notifier
}

// New returns a function that satisfies echo.HandlerFunc interface
// It can be used with Use() methods.
func New(n *gobrake.Notifier) echo.MiddlewareFunc {
	return (&handler{
		notifier: n,
	}).handle
}

func (h *handler) handle(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		if h.notifier == nil {
			log.Println("airbrake notifier not defined")
			return next(c)
		}
		err := next(c)
		_, metric := gobrake.NewRouteMetric(context.TODO(), c.Request().Method, c.Path())

		metric.StatusCode = c.Response().Status
		_ = h.notifier.Routes.Notify(context.TODO(), metric)
		return err
	}
}
