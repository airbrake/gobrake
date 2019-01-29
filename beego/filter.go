package beego

import (
	"time"

	"github.com/airbrake/gobrake"

	"github.com/astaxie/beego"
	"github.com/astaxie/beego/context"
)

func beforeExecFunc() func(c *context.Context) {
	return func(c *context.Context) {
		c.Input.SetData("StartTime", time.Now())
	}
}

func afterExecFunc(notifier *gobrake.Notifier) func(c *context.Context) {
	return func(c *context.Context) {
		routerPattern, ok := c.Input.GetData("RouterPattern").(string)
		if !ok {
			return
		}

		statusCode := c.Output.Status
		if statusCode == 0 {
			statusCode = 200
		}

		startTime, ok := c.Input.GetData("StartTime").(time.Time)
		if !ok {
			return
		}

		notifier.Routes.Notify(&gobrake.RouteInfo{
			Method:     c.Input.Method(),
			Route:      routerPattern,
			StatusCode: statusCode,
			Start:      startTime,
			End:        time.Now(),
		})
	}
}

func InsertAirbrakeFilters(notifier *gobrake.Notifier) {
	beego.InsertFilter("*", beego.BeforeExec, beforeExecFunc(), false)
	beego.InsertFilter("*", beego.AfterExec, afterExecFunc(notifier), false)
}
