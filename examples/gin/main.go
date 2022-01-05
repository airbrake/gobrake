package main

import (
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/airbrake/gobrake/v5"
	ginbrake "github.com/airbrake/gobrake/v5/gin"
	"github.com/gin-gonic/gin"
)

var ProjectId int64 = 999999                               // Insert your Project Id here
var ProjectKey string = "xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx" // Insert your Project Key here

// Define models

type WeatherInfo struct {
	Lat            float32       `json:"lat,omitempty"`
	Lon            float32       `json:"lon,omitempty"`
	Timezone       string        `json:"timezone,omitempty"`
	TimezoneOffset float32       `json:"timezone_offset,omitempty"`
	Current        interface{}   `json:"current,omitempty"`
	Minutely       []interface{} `json:"minutely,omitempty"`
	Hourly         []interface{} `json:"hourly,omitempty"`
	Daily          []interface{} `json:"daily,omitempty"`
}

var availableLocations = []string{
	"austin",
	"pune",
	"santabarbara",
}

var notifier = gobrake.NewNotifierWithOptions(&gobrake.NotifierOptions{
	ProjectId:   ProjectId,
	ProjectKey:  ProjectKey,
	Environment: "production",
})

func main() {

	api := gin.Default()

	// Initialise middlewares
	api.Use(ginbrake.New(notifier))
	api.Use(TokenAuthMiddleware())

	// Initialise routes
	api.GET("/date", getDate)
	api.GET("/locations", getLocations)
	api.GET("/weather/:location", getWeather)

	// Initialise application
	api.Run(":3000")
}

// Define handlers for routes

func getDate(c *gin.Context) {
	secs := time.Now().Unix()
	c.JSON(http.StatusOK, gin.H{"date": secs})
}

func getLocations(c *gin.Context) {
	c.JSON(http.StatusOK, availableLocations)
}

func getWeather(c *gin.Context) {
	var weatherInfo WeatherInfo
	location := strings.TrimSpace(c.Param("location")) // It lets you allow whitespaces but case-sensitive
	if !contains(availableLocations, location) {
		c.JSON(http.StatusNotFound, weatherInfo)
		return
	}
	weatherInfoFile := location + ".json"
	weatherInfoFilePath := filepath.Join("assets", weatherInfoFile)
	// Open our jsonFile
	jsonFile, err := os.Open(weatherInfoFilePath)
	// if we os.Open returns an error then handle it
	if err != nil {
		notifier.Notify(err, nil)
		c.JSON(http.StatusNoContent, weatherInfo)
		return
	}
	byteValue, _ := io.ReadAll(jsonFile)
	json.Unmarshal(byteValue, &weatherInfo)
	// defer the closing of our jsonFile so that we can parse it later on
	defer jsonFile.Close()
	c.JSON(http.StatusOK, weatherInfo)
}

// Function to check whether it is a valid location
func contains(a []string, x string) bool {
	for _, n := range a {
		if x == n {
			return true
		}
	}
	return false
}

func respondWithError(c *gin.Context, code int, message interface{}) {
	c.AbortWithStatusJSON(code, gin.H{"error": message})
}

func TokenAuthMiddleware() gin.HandlerFunc {
	requiredToken := "d4b371692d361869183d92d84caa5edb8835cf7d"

	return func(c *gin.Context) {
		token := c.Request.Header.Get("api-key")

		if token == "" {
			respondWithError(c, http.StatusUnauthorized, "API key required")
			return
		}

		if token != requiredToken {
			respondWithError(c, http.StatusUnauthorized, "Invalid API key")
			return
		}

		c.Next()
	}
}
