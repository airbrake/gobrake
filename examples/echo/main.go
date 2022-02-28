package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	echobrake "github.com/airbrake/gobrake/v5/echo"

	"github.com/airbrake/gobrake/v5"
	"github.com/labstack/echo/v4"
)

// Define models

type Date struct {
	Date int64 `json:"date,omitempty"`
}

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

var ProjectId int64 = 999999                               // Insert your Project Id here
var ProjectKey string = "xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx" // Insert your Project Key here

var notifier = gobrake.NewNotifierWithOptions(&gobrake.NotifierOptions{
	ProjectId:   ProjectId,
	ProjectKey:  ProjectKey,
	Environment: "production",
})

func main() {
	e := echo.New()
	e.Use(echobrake.New(notifier))

	e.GET("/date", getDate)
	e.GET("/locations", getLocations)
	e.GET("/weather/:location", getWeather)

	e.Start(":3000")
}

func getDate(c echo.Context) error {
	var date Date
	date.Date = time.Now().Unix()
	return c.JSON(http.StatusOK, date)
}

func getLocations(c echo.Context) error {
	availableLocations, err := locations()
	if err != nil {
		notifier.Notify(err, nil) // send error to airbrake
		return c.JSON(http.StatusNotFound, availableLocations)
	}
	return c.JSON(http.StatusOK, availableLocations)
}

func getWeather(c echo.Context) error {
	var weatherInfo WeatherInfo
	location := strings.TrimSpace(c.Param("location")) // It lets you allow whitespaces but case-sensitive

	weatherResp, err := checkWeather(location)
	if err != nil {
		notifier.Notify(err, nil) // send error to airbrake
		return c.JSON(http.StatusNotFound, weatherInfo)
	}
	json.Unmarshal(weatherResp, &weatherInfo)
	return c.JSON(http.StatusOK, weatherInfo)
}

func locations() ([]string, error) {
	var locations []string
	weatherInfoFile := "https://airbrake.github.io/weatherapi/locations"
	client := &http.Client{}
	req, err := http.NewRequest("GET", weatherInfoFile, nil)
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
	if resp.StatusCode == 404 {
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
	req, err := http.NewRequest("GET", weatherInfoFile, nil)
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
	if resp.StatusCode == 404 {
		errMsg := fmt.Sprintf("%s weather data not found", location)
		return nil, errors.New(errMsg)
	}
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	return bodyBytes, nil
}
