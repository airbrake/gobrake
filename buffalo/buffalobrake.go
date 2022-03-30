package buffalo

import (
	"errors"
	"net/http"

	"github.com/airbrake/gobrake/v5"
	"github.com/gobuffalo/buffalo"
)

// A Handler is a buffalo middleware that provides integration with
// Airbrake.
type Handler struct {
	Notifier *gobrake.Notifier
}

// New returns a new Airbrake notifier instance. Use the Handle method to wrap
// existing buffalo handlers.
func New(notifier *gobrake.Notifier) (*Handler, error) {
	if notifier == nil {
		return nil, errors.New("airbrake notifier not defined")
	}
	h := Handler{notifier}
	return &h, nil
}

// Handle works as a middleware that wraps an existing buffalo.Handler and sends route performance stats
func (h *Handler) Handle(next buffalo.Handler) buffalo.Handler {
	return func(c buffalo.Context) error {
		_, metric := gobrake.NewRouteMetric(c, c.Request().Method, c.Value("current_route").(buffalo.RouteInfo).Path)

		err := next(c)
		ws, ok := c.Response().(*buffalo.Response)
		if !ok {
			ws = &buffalo.Response{ResponseWriter: c.Response()}
			ws.Status = http.StatusOK
		}
		metric.StatusCode = ws.Status
		_ = h.Notifier.Routes.Notify(c, metric)
		return err
	}
}
