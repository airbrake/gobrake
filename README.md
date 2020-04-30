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

Example
-------

This is the minimal example that you can use to test Gobrake with your project.

```go
package main

import (
	"errors"

	"github.com/airbrake/gobrake/v4"
)

var airbrake = gobrake.NewNotifierWithOptions(&gobrake.NotifierOptions{
	ProjectId:  105138,
	ProjectKey: "fd04e13d806a90f96614ad8e529b2822",
})

func main() {
	defer airbrake.Close()

	airbrake.Notify(errors.New("operation failed"), nil)
}
```

Configuration
-------------

There are two ways to configure Gobrake: quick and dirty & full.

### Quick and dirty configuration

To configure a notifier quickly, you can call `gobrake.NewNotifier`, which
accepts only two arguments: project id and project key. All of the other options
will be set to default values.

```go
airbrake := gobrake.NewNotifier(105138, "fd04e13d806a90f96614ad8e529b2822")
```

### Full configuration

Full configuration is done through `gobrake.NotifierOptions` struct, which you
are supposed to pass to `gobrake.NewNotifierWithOptions`. This way is much more
flexible as it allows configuring all aspects of the notifier. It's the
recommended way to configure your notifier.

```go
airbrake := gobrake.NewNotifierWithOptions(&gobrake.NotifierOptions{
	ProjectId:  105138,
	ProjectKey: "fd04e13d806a90f96614ad8e529b2822",
})
```

#### ProjectId & ProjectKey

You **must** set both `ProjectId` & `ProjectKey`.

To find your `ProjectId` (`int64`) and `ProjectKey` (`string`) navigate to your
project's _General Settings_ and copy the values from the right sidebar.

![][project-idkey]

#### Host

By default, it is set to `https://api.airbrake.io`. A `host` (`string`) is a web
address containing a scheme ("http" or "https"), a host and a port. You can omit
the port (80 will be assumed) and the scheme ("https" will be assumed).

```go
opts := gobrake.NotifierOptions{
	Host: "http://localhost:8080/api/",
}
```

#### Environment

Configures the environment the application is running in. Helps Airbrake
dashboard to distinguish between exceptions occurring in different
environments. By default, it's not set. Expects `string` type.

```go
opts := gobrake.NotifierOptions{
	Environment: "production",
}
```

#### Revision

Specifies current version control revision. If your app runs on Heroku, its
value will be defaulted to `SOURCE_VERSION` environment variable. For non-Heroku
apps this option is not set. Expects `string` type.

```go
opts := gobrake.NotifierOptions{
	Revision: "d34db33f",
}
```

#### KeysBlacklist

Specifies which keys in the payload (parameters, session data, environment data,
etc) should be filtered. Before sending an error, filtered keys will be
substituted with the `[Filtered]` label.

By default, `password` and `secret` are filtered out. `string` and
`*regexp.Regexp` types are permitted.

```go
// String keys.
secrets := []string{"mySecretKey"}

// OR regexp keys
// secrets := []*regexp.Regexp{regexp.MustCompile("mySecretKey")}

blacklist := make([]interface{}, len(secrets))
for i, v := range secrets {
	blacklist[i] = v
}

opts := gobrake.NotifierOptions{
	KeysBlacklist: blacklist,
}
```

#### DisableCodeHunks

Controls code hunk collection. Code hunks are lines of code surrounding each
backtrace frame. By default, it's set to `false`. Expects `bool` type.

```go
opts := gobrake.NotifierOptions{
	DisableCodeHunks: true,
}
```

#### HTTPClient

HTTP client that is used to send data to Airbrake API. Expects `*http.Client`
type. Normally, you shouldn't configure it.

```go
opts := gobrake.NotifierOptions{
	HTTPClient: &http.Client{
		Timeout: 10 * time.Second,
	},
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

License
-------

The project uses the MIT License. See LICENSE.md for details.

[arthur-go]: http://f.cl.ly/items/3J3h1L05222X3o1w2l2L/golang.jpg
[airbrake.io]: https://airbrake.io
[gobrake-github]: https://github.com/airbrake/gobrake
[docs]: https://pkg.go.dev/github.com/airbrake/gobrake?tab=doc
[beego]: https://beego.me
[gin]: https://github.com/gin-gonic/gin
[semver2]: http://semver.org/spec/v2.0.0.html
[go-mod]: https://github.com/golang/go/wiki/Modules
[project-idkey]: https://s3.amazonaws.com/airbrake-github-assets/airbrake-ruby/project-id-key.png
