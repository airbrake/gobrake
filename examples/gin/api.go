package main

import (
	"flag"

	"github.com/airbrake/gobrake"

	"github.com/gin-gonic/gin"
)

var env = flag.String("env", "development", "environment, e.g. development or production")
var host = flag.String("airbrake_host", "", "airbrake host")
var projectId = flag.Int64("airbrake_project_id", 0, "airbrake project ID")
var projectKey = flag.String("airbrake_project_key", "", "airbrake project key")

func main() {
	flag.Parse()

	api := gin.Default()

	notifier := gobrake.NewNotifierWithOptions(&gobrake.NotifierOptions{
		ProjectId:   *projectId,
		ProjectKey:  *projectKey,
		Host:        *host,
		Environment: *env,
	})
	routeStatMiddleware := NewAirbrakeMiddleware(api, notifier)
	api.Use(routeStatMiddleware)

	api.GET("/hello/:name", hello)
	api.GET("/ping", ping)
	api.Run(":8080")
}

func hello(c *gin.Context) {
	name := c.Param("name")
	c.String(200, "Hello %s", name)
}

func ping(c *gin.Context) {
	c.String(200, "Ping")
}
