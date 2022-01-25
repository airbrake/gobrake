package fiber

import (
	"context"
	"errors"

	"github.com/airbrake/gobrake/v5"

	"github.com/gofiber/fiber/v2"
)

// New returns a function that satisfies fiber.Handler interface
func New(notifier *gobrake.Notifier) fiber.Handler {
	return func(c *fiber.Ctx) error {
		if notifier == nil {
			return errors.New("airbrake notifier not defined")
		}
		_, metric := gobrake.NewRouteMetric(context.TODO(), c.Route().Method, c.Route().Path)
		err := c.Next()
		metric.StatusCode = c.Response().StatusCode()
		_ = notifier.Routes.Notify(context.TODO(), metric)
		return err
	}
}
