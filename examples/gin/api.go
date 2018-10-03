package main

import (
	"flag"
	"time"

	"github.com/airbrake/gobrake"
	ginbrake "github.com/airbrake/gobrake/gin"

	"github.com/gin-gonic/gin"
)

var env = flag.String("env", "development", "environment, e.g. development or production")
var projectId = flag.Int64("airbrake_project_id", 0, "airbrake project ID")
var projectKey = flag.String("airbrake_project_key", "", "airbrake project key")

func main() {
	flag.Parse()

	api := gin.Default()

	notifier := gobrake.NewNotifierWithOptions(&gobrake.NotifierOptions{
		ProjectId:   *projectId,
		ProjectKey:  *projectKey,
		Environment: *env,
	})
	api.Use(ginbrake.NewMiddleware(api, notifier))

	api.GET("/hello/:name", hello)
	api.GET("/ping", ping)
	api.Run(":8080")
}

func hello(c *gin.Context) {
	name := c.Param("name")
	c.String(200, "Hello %s", name)
}

func ping(c *gin.Context) {
	time.Sleep(100 * time.Millisecond)
	c.String(200, "Ping")
}
