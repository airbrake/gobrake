Gobrake
=======

[![Build Status](https://travis-ci.org/airbrake/gobrake.svg?branch=v2)](https://travis-ci.org/airbrake/gobrake)

![Gobrake][arthur-go]

* [Gobrake README][gobrake-github]
* [pkg.go.dev documentation][docs]

Introduction
------------

_Gobrake_ is the official notifier library for [Airbrake][airbrake.io] for the
Go programming language, the leading exception reporting service. Gobrake
provides a minimalist API that enables the ability to send _any_ Go error or
panic to the Airbrake dashboard. The library is extremely lightweight, with
minimal overhead.

Key features
------------

* Simple, consistent and easy-to-use library API
* Asynchronous exception reporting
* Flexible configuration options
* Support for environments
* Filters support (filter out sensitive or unwanted data that shouldn't be sent)
* Ability to ignore certain errors
* SSL support (all communication with Airbrake is encrypted by default)
* Panic reporting support
* Severity support
* Support for code hunks (lines of code surrounding each backtrace frame)
* Automatic deploy tracking
* Performance monitoring features such as HTTP route statistics, SQL queries,
  and Job execution statistics
* Integrations with [Beego][beego] and [Gin][gin]
* Last but not least, we follow [semantic versioning 2.0.0][semver2]

Installation
------------

### Go modules

Gobrake can be installed like any other Go package that supports [Go
modules][go-mod].

#### Installing in an existing project

Just `go get` the library:

```sh
go get github.com/airbrake/gobrake/v4
```

#### Installing in a new project

Create a new directory, initialize a new module and `go get` the library:

```sh
mkdir airbrake_example && cd airbrake_example
go mod init airbrake_example
go get github.com/airbrake/gobrake/v4
```

# Example

``` go
package main

import (
    "errors"

    "github.com/airbrake/gobrake/v4"
)

var airbrake = gobrake.NewNotifierWithOptions(&gobrake.NotifierOptions{
    ProjectId: 123456,
    ProjectKey: "FIXME",
    Environment: "production",
})

func init() {
    airbrake.AddFilter(func(notice *gobrake.Notice) *gobrake.Notice {
        notice.Params["user"] = map[string]string{
            "id": "1",
            "username": "johnsmith",
            "name": "John Smith",
        }
        return notice
    })
}

func main() {
    defer airbrake.Close()
    defer airbrake.NotifyOnPanic()

    airbrake.Notify(errors.New("operation failed"), nil)
}
```

## Ignoring notices

``` go
airbrake.AddFilter(func(notice *gobrake.Notice) *gobrake.Notice {
    if notice.Context["environment"] == "development" {
        // Ignore notices in development environment.
        return nil
    }
    return notice
})
```

## Setting severity

[Severity](https://airbrake.io/docs/airbrake-faq/what-is-severity/) allows
categorizing how severe an error is. By default, it's set to `error`. To
redefine severity, simply overwrite `context/severity` of a notice object. For
example:

``` go
notice := airbrake.NewNotice("operation failed", nil, 0)
notice.Context["severity"] = "critical"
airbrake.Notify(notice, nil)
```

## Logging

You can use [glog fork](https://github.com/airbrake/glog) to send your logs to Airbrake.

## Sending routes stats

In order to collect some basic routes stats you can instrument your application
using `notifier.Routes.Notify` API. We also have prepared HTTP middleware examples for [Gin](examples/gin) and
[Beego](examples/beego).  Here is an example using the net/http middleware.

``` go
package main

import (
  "fmt"
  "net/http"

  "github.com/airbrake/gobrake"
)

// Airbrake is used to report errors and track performance
var Airbrake = gobrake.NewNotifierWithOptions(&gobrake.NotifierOptions{
  ProjectId:   123123,              // <-- Fill in this value
  ProjectKey:  "YourProjectAPIKey", // <-- Fill in this value
  Environment: "Production",
})

func indexHandler(w http.ResponseWriter, req *http.Request) {
  fmt.Fprintf(w, "Hello, There!")
}

func main() {
  fmt.Println("Server listening at http://localhost:5555/")
  // Wrap the indexHandler with Airbrake Performance Monitoring middleware:
  http.HandleFunc(airbrakePerformance("/", indexHandler))
  http.ListenAndServe(":5555", nil)
}

func airbrakePerformance(route string, h http.HandlerFunc) (string, http.HandlerFunc) {
  handler := http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
    ctx := req.Context()
    ctx, routeMetric := gobrake.NewRouteMetric(ctx, req.Method, route) // Starts the timing
    arw := newAirbrakeResponseWriter(w)

    h.ServeHTTP(arw, req)

    routeMetric.StatusCode = arw.statusCode
    Airbrake.Routes.Notify(ctx, routeMetric) // Stops the timing and reports
    fmt.Printf("code: %v, method: %v, route: %v\n", arw.statusCode, req.Method, route)
  })

  return route, handler
}

type airbrakeResponseWriter struct {
  http.ResponseWriter
  statusCode int
}

func newAirbrakeResponseWriter(w http.ResponseWriter) *airbrakeResponseWriter {
  // Returns 200 OK if WriteHeader isn't called
  return &airbrakeResponseWriter{w, http.StatusOK}
}

func (arw *airbrakeResponseWriter) WriteHeader(code int) {
  arw.statusCode = code
  arw.ResponseWriter.WriteHeader(code)
}
```


To get more detailed timing you can wrap important blocks of code into spans. For example, you can create 2 spans `sql` and `http` to measure timing of specific operations:

``` go
metric := &gobrake.RouteMetric{
    Method: c.Request.Method,
    Route:  routeName,
    StartTime:  time.Now(),
}

ctx, span := metric.Start(ctx, "sql")
users, err := fetchUser(ctx, userID)
span.Finish()

ctx, span = metric.Start(ctx, "http")
resp, err := http.Get("http://example.com/")
span.Finish()

metric.StatusCode = http.StatusOK
notifier.Routes.Notify(ctx, metric)
```

You can also collect stats about individual SQL queries performance using following API:

``` go
notifier.Queries.Notify(&gobrake.QueryInfo{
    Query:     "SELECT * FROM users WHERE id = ?", // query must be normalized
    Func:      "fetchUser", // optional
    File:      "models/user.go", // optional
    Line:      123, // optional
    StartTime: startTime,
    EndTime:   time.Now(),
})
```

## Sending queue stats

``` go
metric := &gobrake.QueueMetric{
    Queue: "my-queue-name",
    StartTime:  time.Now(),
}

ctx, span := metric.Start(ctx, "sql")
users, err := fetchUser(ctx, userID)
span.Finish()

ctx, span = metric.Start(ctx, "http")
resp, err := http.Get("http://example.com/")
span.Finish()

notifier.Queues.Notify(ctx, metric)
```

[arthur-go]: http://f.cl.ly/items/3J3h1L05222X3o1w2l2L/golang.jpg
[airbrake.io]: https://airbrake.io
[gobrake-github]: https://github.com/airbrake/gobrake
[docs]: https://pkg.go.dev/github.com/airbrake/gobrake?tab=doc
[beego]: https://beego.me
[gin]: https://github.com/gin-gonic/gin
[semver2]: http://semver.org/spec/v2.0.0.html
[go-mod]: https://github.com/golang/go/wiki/Modules
