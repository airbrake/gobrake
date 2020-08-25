package beego

import (
	goctx "context"

	"github.com/airbrake/gobrake/v5"

	"github.com/astaxie/beego"
	"github.com/astaxie/beego/context"
)

const abMetricKey = "ab_metric"

func beforeExecFunc() func(c *context.Context) {
	return func(c *context.Context) {
		routerPattern, ok := c.Input.GetData("RouterPattern").(string)
		if !ok {
			return
		}

		_, metric := gobrake.NewRouteMetric(goctx.TODO(), c.Input.Method(), routerPattern)
		c.Input.SetData(abMetricKey, metric)
	}
}

func afterExecFunc(notifier *gobrake.Notifier) func(c *context.Context) {
	return func(c *context.Context) {
		metric, ok := c.Input.GetData(abMetricKey).(*gobrake.RouteMetric)
		if !ok {
			return
		}

		metric.StatusCode = c.Output.Status
		if metric.StatusCode == 0 {
			metric.StatusCode = 200
		}

		_ := notifier.Routes.Notify(goctx.TODO(), metric)
	}
}

func InsertAirbrakeFilters(notifier *gobrake.Notifier) {
	beego.InsertFilter("*", beego.BeforeExec, beforeExecFunc(), false)
	beego.InsertFilter("*", beego.AfterExec, afterExecFunc(notifier), false)
}
