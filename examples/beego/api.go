package main

import (
	"flag"

	"github.com/airbrake/gobrake/v5"
	beegobrake "github.com/airbrake/gobrake/v5/beego"
	"github.com/airbrake/gobrake/v5/examples/beego/controllers"

	"github.com/astaxie/beego"
)

var env = flag.String("env", "development", "environment, e.g. development or production")
var projectId = flag.Int64("airbrake_project_id", 0, "airbrake project ID")
var projectKey = flag.String("airbrake_project_key", "", "airbrake project key")

func main() {
	flag.Parse()

	notifier := gobrake.NewNotifierWithOptions(&gobrake.NotifierOptions{
		ProjectId:   *projectId,
		ProjectKey:  *projectKey,
		Environment: *env,
	})
	beegobrake.InsertAirbrakeFilters(notifier)

	beego.Router("/ping", &controllers.PingController{})
	beego.Router("/hello/:name", &controllers.HelloController{})
	beego.Run()
}
