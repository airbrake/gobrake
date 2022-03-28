package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/airbrake/gobrake/v5"
	fasthttpbrake "github.com/airbrake/gobrake/v5/fasthttp"
	"github.com/valyala/fasthttp"
)

// Define models

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

var ProjectId int64 = 999999                               // Insert your Project Id here
var ProjectKey string = "xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx" // Insert your Project Key here

var notifier = gobrake.NewNotifierWithOptions(&gobrake.NotifierOptions{
	ProjectId:   ProjectId,
	ProjectKey:  ProjectKey,
	Environment: "production",
})

func main() {

	weatherRoute, _ := regexp.Compile(`(weather\/)?([^\/\s]+\/)(.*)`)
	fastHTTPHandler := func(ctx *fasthttp.RequestCtx) {
		switch {
		case string(ctx.Path()) == "/date":
			getDate(ctx)
		case string(ctx.Path()) == "/locations":
			getLocations(ctx)
		case weatherRoute.MatchString(string(ctx.Path())):
			getWeather(ctx)
		default:
			defaultHandler(ctx)
		}
	}
	fmt.Println("Server listening at http://localhost:3000/")
	if err := fasthttp.ListenAndServe(":3000", fasthttpbrake.New(notifier, fastHTTPHandler)); err != nil {
		log.Fatalf("Error in ListenAndServe: %s", err)
	}
}

func getDate(ctx *fasthttp.RequestCtx) {
	ctx.SetContentType("application/json")
	ctx.SetStatusCode(fasthttp.StatusOK)
	var date Date
	date.Date = time.Now().Unix()
	json.NewEncoder(ctx).Encode(date)
}

func getLocations(ctx *fasthttp.RequestCtx) {
	ctx.SetContentType("application/json")
	availableLocations, err := locations()
	json.NewEncoder(ctx).Encode(availableLocations)
	if err != nil {
		notifier.Notify(err, nil) // send error to airbrake
		ctx.SetStatusCode(fasthttp.StatusNotFound)
		return
	}
}

func getWeather(ctx *fasthttp.RequestCtx) {
	var weatherInfo WeatherInfo
	ctx.SetContentType("application/json")
	location := strings.TrimPrefix(string(ctx.Path()), "/weather/")
	location = strings.TrimSpace(location) // It lets you allow whitespaces but case-sensitive
	weatherResp, err := checkWeather(location)
	if err != nil {
		notifier.Notify(err, nil) // send error to airbrake
		ctx.SetStatusCode(fasthttp.StatusNotFound)
		json.NewEncoder(ctx).Encode(weatherInfo)
		return
	}
	ctx.SetStatusCode(fasthttp.StatusOK)
	json.Unmarshal(weatherResp, &weatherInfo)
	json.NewEncoder(ctx).Encode(weatherInfo)
}

func defaultHandler(ctx *fasthttp.RequestCtx) {
	ctx.Error("Unsupported path", fasthttp.StatusNotFound)
}

func locations() ([]string, error) {
	var locations []string
	weatherInfoFile := "https://airbrake.github.io/weatherapi/locations"
	client := &http.Client{}
	req, err := http.NewRequest(fasthttp.MethodGet, weatherInfoFile, nil)
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
	if resp.StatusCode == fasthttp.StatusNotFound {
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
	req, err := http.NewRequest(fasthttp.MethodGet, weatherInfoFile, nil)
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
	if resp.StatusCode == fasthttp.StatusNotFound {
		errMsg := fmt.Sprintf("%s weather data not found", location)
		return nil, errors.New(errMsg)
	}
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	return bodyBytes, nil
}
