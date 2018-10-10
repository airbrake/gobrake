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
		startTime, ok := c.Input.GetData("StartTime").(time.Time)
		if !ok {
			return
		}

		routerPattern, ok := c.Input.GetData("RouterPattern").(string)
		if !ok {
			return
		}

		statusCode := c.Output.Status
		if statusCode == 0 {
			statusCode = 200
		}

		dur := time.Since(startTime)
		notifier.IncRequest(c.Input.Method(), routerPattern, statusCode, dur, startTime)
	}
}

func InsertAirbrakeFilters(notifier *gobrake.Notifier) {
	beego.InsertFilter("*", beego.BeforeExec, beforeExecFunc(), false)
	beego.InsertFilter("*", beego.AfterExec, afterExecFunc(notifier), false)
}
