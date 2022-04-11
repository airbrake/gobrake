package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
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

var notifier = gobrake.NewNotifierWithOptions(&gobrake.NotifierOptions{
	ProjectId:   ProjectId,
	ProjectKey:  ProjectKey,
	Environment: "production",
})

func main() {

	api := gin.Default()

	// Initialize middlewares
	api.Use(ginbrake.New(notifier))

	// Initialize routes
	api.GET("/date", getDate)
	api.GET("/locations", getLocations)
	api.GET("/weather/:location", getWeather)

	// Initialize application
	api.Run(":3000")
}

// Define handlers for routes

func getDate(c *gin.Context) {
	secs := time.Now().Unix()
	c.JSON(http.StatusOK, gin.H{"date": secs})
}

func getLocations(c *gin.Context) {
	availableLocations, err := locations()
	if err != nil {
		notifier.Notify(err, nil) // send error to airbrake
		c.JSON(http.StatusNotFound, availableLocations)
		return
	}
	c.JSON(http.StatusOK, availableLocations)
}

func getWeather(c *gin.Context) {
	var weatherInfo WeatherInfo
	location := strings.TrimSpace(c.Param("location")) // It lets you allow whitespaces but case-sensitive
	weatherResp, err := checkWeather(location)
	json.Unmarshal(weatherResp, &weatherInfo)
	if err != nil {
		notifier.Notify(err, nil) // send error to airbrake
		c.JSON(http.StatusNotFound, weatherInfo)
		return
	}
	c.JSON(http.StatusOK, weatherInfo)
}

func locations() ([]string, error) {
	var locations []string
	weatherInfoFile := "https://airbrake.github.io/weatherapi/locations"
	client := &http.Client{}
	req, err := http.NewRequest(http.MethodGet, weatherInfoFile, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Add("Accept", "application/json")
	req.Header.Add("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return nil, errors.New("locations not found")
	}
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	json.Unmarshal(bodyBytes, &locations)
	return locations, nil
}

func checkWeather(location string) ([]byte, error) {
	weatherInfoFile := "https://airbrake.github.io/weatherapi/weather/" + location
	client := &http.Client{}
	req, err := http.NewRequest(http.MethodGet, weatherInfoFile, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Add("Accept", "application/json")
	req.Header.Add("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		errMsg := fmt.Sprintf("%s weather data not found", location)
		return nil, errors.New(errMsg)
	}
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	return bodyBytes, nil
}
