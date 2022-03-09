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
	beegobrake "github.com/airbrake/gobrake/v5/beego"
	"github.com/beego/beego/v2/server/web"
)

type Date struct {
	Date int64 `json:"date"`
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

type Controller struct {
	web.Controller
}

var ProjectId int64 = 999999                               // Insert your Project Id here
var ProjectKey string = "xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx" // Insert your Project Key here

var notifier = gobrake.NewNotifierWithOptions(&gobrake.NotifierOptions{
	ProjectId:   ProjectId,
	ProjectKey:  ProjectKey,
	Environment: "production",
})

func main() {
	ctrl := &Controller{}
	web.InsertFilterChain("/*", beegobrake.New(notifier))
	web.Router("/date", ctrl, "get:GetDate")
	web.Router("/locations", ctrl, "get:GetLocations")
	web.Router("/weather/:location", ctrl, "get:GetWeather")

	web.Run("127.0.0.1:3000")
}

func (ctrl *Controller) GetDate() {
	ctrl.JSONResp(&Date{Date: time.Now().Unix()})
}

func (ctrl *Controller) GetLocations() {
	availableLocations, err := locations()
	if err != nil {
		notifier.Notify(err, nil) // send error to airbrake
		ctrl.Ctx.ResponseWriter.WriteHeader(http.StatusNotFound)
		ctrl.JSONResp(availableLocations)
		return
	}
	ctrl.JSONResp(availableLocations)
}

func (ctrl *Controller) GetWeather() {
	var weatherInfo WeatherInfo
	location := strings.TrimSpace(ctrl.Ctx.Input.Param(":location")) // It lets you allow whitespaces but case-sensitive
	weatherResp, err := checkWeather(location)
	if err != nil {
		notifier.Notify(err, nil) // send error to airbrake
		ctrl.Ctx.ResponseWriter.WriteHeader(http.StatusNotFound)
		ctrl.JSONResp(weatherInfo)
		return
	}
	json.Unmarshal(weatherResp, &weatherInfo)
	ctrl.JSONResp(weatherInfo)
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
