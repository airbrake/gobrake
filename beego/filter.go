package beego

import (
	"github.com/airbrake/gobrake"

	"github.com/astaxie/beego"
	"github.com/astaxie/beego/context"
)

const abTraceKey = "ab_trace"

func beforeExecFunc() func(c *context.Context) {
	return func(c *context.Context) {
		routerPattern, ok := c.Input.GetData("RouterPattern").(string)
		if !ok {
			return
		}

		_, trace := gobrake.NewRouteTrace(nil, c.Input.Method(), routerPattern)
		c.Input.SetData(abTraceKey, trace)
	}
}

func afterExecFunc(notifier *gobrake.Notifier) func(c *context.Context) {
	return func(c *context.Context) {
		trace, ok := c.Input.GetData(abTraceKey).(*gobrake.RouteTrace)
		if !ok {
			return
		}

		trace.StatusCode = c.Output.Status
		if trace.StatusCode == 0 {
			trace.StatusCode = 200
		}

		notifier.Routes.Notify(nil, trace)
	}
}

func InsertAirbrakeFilters(notifier *gobrake.Notifier) {
	beego.InsertFilter("*", beego.BeforeExec, beforeExecFunc(), false)
	beego.InsertFilter("*", beego.AfterExec, afterExecFunc(notifier), false)
}
