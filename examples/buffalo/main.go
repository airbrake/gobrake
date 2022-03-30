package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/airbrake/gobrake/v5"
	buffalobrake "github.com/airbrake/gobrake/v5/buffalo"
	"github.com/gobuffalo/buffalo"
	"github.com/gobuffalo/buffalo/render"
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

var r *render.Engine

func main() {
	app := App()
	log.Fatal(app.Serve())
}

func App() *buffalo.App {
	app := buffalo.New(buffalo.Options{})
	airbrake, err := buffalobrake.New(notifier)
	if err != nil {
		log.Fatal(err)
	}
	app.Use(airbrake.Handle)
	app.GET("/date", GetDate)
	app.GET("/locations", getLocations)
	app.GET("/weather/{location}", getWeather)
	return app
}

func GetDate(c buffalo.Context) error {
	// fmt.Println(c.Request().Response.StatusCode)
	var date Date
	date.Date = time.Now().Unix()
	return c.Render(http.StatusOK, r.JSON(date))
}

func getLocations(c buffalo.Context) error {
	availableLocations, err := locations()
	if err != nil {
		notifier.Notify(err, nil) // send error to airbrake
		return c.Render(http.StatusNotFound, r.JSON(availableLocations))
	}
	return c.Render(http.StatusOK, r.JSON(availableLocations))
}

func getWeather(c buffalo.Context) error {
	var weatherInfo WeatherInfo
	location := strings.TrimSpace(c.Param("location")) // It lets you allow whitespaces but case-sensitive
	weatherResp, err := checkWeather(location)
	json.Unmarshal(weatherResp, &weatherInfo)
	if err != nil {
		notifier.Notify(err, nil) // send error to airbrake
		return c.Render(http.StatusNotFound, r.JSON(weatherInfo))
	}
	return c.Render(http.StatusOK, r.JSON(weatherInfo))
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
