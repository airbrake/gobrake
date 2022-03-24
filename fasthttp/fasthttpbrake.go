package fasthttp

import (
	"context"

	"github.com/airbrake/gobrake/v5"
	"github.com/valyala/fasthttp"
)

// New returns a function that satisfies fasthttp.RequestHandler interface
func New(notifier *gobrake.Notifier, next fasthttp.RequestHandler) fasthttp.RequestHandler {
	return func(ctx *fasthttp.RequestCtx) {
		_, metric := gobrake.NewRouteMetric(context.TODO(), string(ctx.Method()), string(ctx.Path()))

		next(ctx)

		metric.StatusCode = ctx.Response.Header.StatusCode()
		_ = notifier.Routes.Notify(context.TODO(), metric)

	}
}
