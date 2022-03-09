package beego

import (
	goctx "context"
	"log"

	"github.com/airbrake/gobrake/v5"

	"github.com/beego/beego/v2/server/web"
	"github.com/beego/beego/v2/server/web/context"
)

func New(notifier *gobrake.Notifier) web.FilterChain {
	return func(next web.FilterFunc) web.FilterFunc {
		return func(ctx *context.Context) {
			if notifier == nil {
				log.Println("airbrake notifier not defined")
				next(ctx)
				return
			}

			routerPattern := ""
			ctrl := web.BeeApp.Handlers
			if rt, found := ctrl.FindRouter(ctx); found {
				routerPattern = rt.GetPattern()
			} else {
				next(ctx)
				return
			}
			_, metric := gobrake.NewRouteMetric(goctx.TODO(), ctx.Input.Method(), routerPattern)
			next(ctx)
			metric.StatusCode = ctx.ResponseWriter.Status
			_ = notifier.Routes.Notify(goctx.TODO(), metric)

		}
	}
}
