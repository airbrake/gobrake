package iris

import (
	"context"
	"log"

	"github.com/airbrake/gobrake/v5"
	"github.com/kataras/iris/v12"
)

// New returns a function that satisfies iris.Handler interface
// It can be used with Use() methods.
func New(n *gobrake.Notifier) iris.Handler {
	return func(ctx iris.Context) {
		ctx.Next()
		if n == nil {
			log.Println("airbrake notifier not defined")
			return
		}

		_, metric := gobrake.NewRouteMetric(context.TODO(), ctx.Method(), ctx.GetCurrentRoute().Path())

		metric.StatusCode = ctx.GetStatusCode()
		_ = n.Routes.Notify(context.TODO(), metric)

	}
}
