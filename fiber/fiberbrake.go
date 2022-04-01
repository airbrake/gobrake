package fiber

import (
	"context"
	"log"

	"github.com/airbrake/gobrake/v5"

	"github.com/gofiber/fiber/v2"
)

// New returns a function that satisfies fiber.Handler interface
func New(notifier *gobrake.Notifier) fiber.Handler {
	return func(c *fiber.Ctx) error {
		if notifier == nil {
			log.Println("airbrake notifier not defined")
			return c.Next()
		}

		// Starts the timer.
		_, metric := gobrake.NewRouteMetric(context.TODO(), c.Route().Method, c.Route().Path)
		err := c.Next()

		// capture the status code and resolved Route
		metric.StatusCode = c.Response().StatusCode()
		metric.Route = c.Route().Path

		// Send to Airbrake
		_ = notifier.Routes.Notify(context.TODO(), metric)
		return err
	}
}
