# Fiber integration

This is an example of a basic Fiber app with Airbrake middleware that reports performance data (route stats).

## How to run Example API

Insert your project ID and project key in the `main.go` file. You can find these values on the settings page for your project.

Initialise mod file

```bash
go mod init
go mod tidy
```

Run go application

```bash
go run main.go
```

The example application provides three GET endpoints:

1. `/date` - gets the system date from the server
2. `/locations` - gets the supported locations for use with the `/weather` endpoint
3. `/weather/{locationName}` - gets the weather for a location; valid values for `locationName` can be found using the `/locations` endpoint

Use the cURL commands below to interact with the endpoints.

```bash
curl "http://localhost:3000/date"
curl "http://localhost:3000/locations"
curl "http://localhost:3000/weather/{austin/pune/santabarbara}"
```

To see how Airbrake error monitoring works, use an unsupported location, e.g. `boston`: `curl "http://localhost:3000/weather/boston"`.
After issuing this request, the service will respond with a `404 Not Found` error.  Visit the Airbrake dashboard to see the error captured there.

Once you call the API endpoints, view the Airbrake errors and performance dashboards for your project.
