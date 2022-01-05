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
	fiberbrake "github.com/airbrake/gobrake/v5/fiber"
	"github.com/gofiber/fiber/v2"
)

var ProjectId int64 = 999999                               // Insert your Project Id here
var ProjectKey string = "xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx" // Insert your Project Key here

var notifier = gobrake.NewNotifierWithOptions(&gobrake.NotifierOptions{
	ProjectId:   ProjectId,
	ProjectKey:  ProjectKey,
	Environment: "production",
})

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

func main() {

	app := fiber.New()
	app.Use(fiberbrake.New(notifier))
	// Initialise routes
	app.Get("/date", getDate)
	app.Get("/locations", getLocations)
	app.Get("/weather/:location", getWeather)
	app.Listen(":3000")
	defer notifier.Close()
}

func getDate(c *fiber.Ctx) error {
	var date Date
	date.Date = time.Now().Unix()
	return c.Status(http.StatusOK).JSON(date)
}

func getLocations(c *fiber.Ctx) error {
	availableLocations, err := locations()
	if err != nil {
		notifier.Notify(err, nil) // send error to airbrake
		return c.Status(http.StatusNotFound).JSON(availableLocations)
	}
	return c.Status(http.StatusOK).JSON(availableLocations)
}

func getWeather(c *fiber.Ctx) error {
	var weatherInfo WeatherInfo
	location := strings.TrimSpace(c.Params("location")) // It lets you allow whitespaces but case-sensitive

	weatherResp, err := checkWeather(location)
	if err != nil {
		notifier.Notify(err, nil) // send error to airbrake
		return c.Status(http.StatusNotFound).JSON(weatherInfo)
	}
	json.Unmarshal(weatherResp, &weatherInfo)
	return c.Status(http.StatusOK).JSON(weatherInfo)
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
