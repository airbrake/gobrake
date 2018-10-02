package main

import (
	"github.com/airbrake/gobrake/cmd/api/route_stat"

	"github.com/gin-gonic/gin"
)

func main() {
	api := gin.Default()

	routeStatMiddleware := route_stat.NewAirbrakeMiddleware(api)
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
