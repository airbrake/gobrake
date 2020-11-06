# Gobrake

[![Circle Build Status](https://circleci.com/gh/airbrake/gobrake.svg?style=shield)](https://circleci.com/gh/airbrake/gobrake)
[![PkgGoDev](https://pkg.go.dev/badge/airbrake/gobrake)][docs]

![Gobrake][arthur-go]

* [Gobrake README][gobrake-github]
* [pkg.go.dev documentation][docs]

## Introduction

_Gobrake_ is the official notifier library for [Airbrake][airbrake.io] for the
Go programming language. Gobrake provides a minimalist API that enables the
ability to send _any_ Go error or panic to the Airbrake dashboard. The library
is extremely lightweight, with minimal overhead.

## Key features

* Simple, consistent and easy-to-use library API
* Asynchronous exception reporting
* Flexible configuration options
* Support for environments
* Add extra context to errors before reporting them
* Filters support (filter out sensitive or unwanted data that shouldn't be sent)
* Ignore errors based on class, message, status, file, or any other filter
* SSL support (all communication with Airbrake is encrypted by default)
* Notify Airbrake on panics
* Set error severity to control notification thresholds
* Support for code hunks (lines of code surrounding each backtrace frame)
* Automatic deploy tracking
* Performance monitoring features such as HTTP route statistics, SQL queries,
  and Job execution statistics
* Integrations with [Beego][beego], [Gin][gin] and [Negroni][negroni]
* Last but not least, we follow [semantic versioning 2.0.0][semver2]

## Installation

### Go modules

Gobrake can be installed like any other Go package that supports [Go
modules][go-mod].

#### Installing in an existing project

Just `go get` the library:

```sh
go get github.com/airbrake/gobrake/v5
```

#### Installing in a new project

Create a new directory, initialize a new module and `go get` the library:

```sh
mkdir airbrake_example && cd airbrake_example
go mod init airbrake_example
go get github.com/airbrake/gobrake/v5
```

## Example

This is the minimal example that you can use to test Gobrake with your project.

```go
package main

import (
	"errors"

	"github.com/airbrake/gobrake/v5"
)

var airbrake = gobrake.NewNotifierWithOptions(&gobrake.NotifierOptions{
	ProjectId: 105138,
	ProjectKey: "fd04e13d806a90f96614ad8e529b2822",
	Environment: "production",
})

func main() {
	defer airbrake.Close()

	airbrake.Notify(errors.New("operation failed"), nil)
}
```

## Configuration

Configuration is done through the `gobrake.NotifierOptions` struct, which you
are supposed to pass to `gobrake.NewNotifierWithOptions`.

```go
airbrake := gobrake.NewNotifierWithOptions(&gobrake.NotifierOptions{
	ProjectId: 105138,
	ProjectKey: "fd04e13d806a90f96614ad8e529b2822",
	Environment: "production",
})
```

### NotifierOptions

#### ProjectId & ProjectKey

You **must** set both `ProjectId` & `ProjectKey`.


To find your `ProjectId` (`int64`) and `ProjectKey` (`string`) navigate to your
project's _Settings_ and copy the values from the right sidebar.

![][project-idkey]

#### Environment

Configures the environment the application is running in. It helps Airbrake
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

#### KeysBlocklist

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

blocklist := make([]interface{}, len(secrets))
for i, v := range secrets {
	blocklist[i] = v
}

opts := gobrake.NotifierOptions{
	KeysBlocklist: blocklist,
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

#### Host

By default, it is set to `https://api.airbrake.io`. A `host` (`string`) is a web
address containing a scheme ("http" or "https"), a host and a port. You can omit
the port (80 will be assumed) and the scheme ("https" will be assumed).

```go
opts := gobrake.NotifierOptions{
	Host: "http://localhost:8080/api/",
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

## API

For complete API description please follow documentation on [pkg.go.dev
documentation][docs].

#### AddFilter

`AddFilter` accepts a callback function which will be executed every time a
`gobrake.Notice` is sent. You can use that for two purposes: filtering of
unwanted or sensitive params or ignoring the whole notice completely.

##### Filtering unwanted params

```go
// Filter out sensitive information such as credit cards.
airbrake.AddFilter(func(n *gobrake.Notice) *gobrake.Notice {
	if _, ok := n.Context["creditCard"] {
		n.Context["creditCard"] = "Filtered"
	}
	return n
})
```

##### Ignoring notices

```go
// Ignore all notices in development.
airbrake.AddFilter(func(n *gobrake.Notice) *gobrake.Notice {
	if n.Context["environment"] == "development" {
		return nil
	}
	return n
})
```

#### Setting severity

[Severity](https://airbrake.io/docs/airbrake-faq/what-is-severity/) allows
categorizing how severe an error is. By default, it's set to `error`. To
redefine severity, simply overwrite `context/severity` of a notice object. For
example:

``` go
notice := airbrake.NewNotice("operation failed", nil, 0)
notice.Context["severity"] = "critical"
airbrake.Notify(notice, nil)
```

### Performance Monitoring

You can read more about our [Performance Monitoring offering in our docs][docs/performance].

#### Sending routes stats

In order to collect routes stats you can instrument your application
using `notifier.Routes.Notify` API.

Below is an example using the net/http middleware. We also have HTTP middleware
examples for [Gin](examples/gin), [Beego](examples/beego) and
[Negroni](examples/negroni).

```go
package main

import (
	"fmt"
	"net/http"

	"github.com/airbrake/gobrake"
)

// Airbrake is used to report errors and track performance
var Airbrake = gobrake.NewNotifierWithOptions(&gobrake.NotifierOptions{
	ProjectId:   <YOUR PROJECT ID>,				// <-- Fill in this value
	ProjectKey:  "<YOUR API KEY>", // <-- Fill in this value
	Environment: "production",
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


To get more detailed timing you can wrap important blocks of code into
spans. For example, you can create 2 spans `sql` and `http` to measure timing of
specific operations:

```go
metric := &gobrake.RouteMetric{
	Method: "GET",
	Route:	"/",
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

#### Sending queries stats

You can also collect stats about individual SQL queries performance using
following API:

```go
notifier.Queries.Notify(
	context.TODO(),
	&gobrake.QueryInfo{
		Query:	   "SELECT * FROM users WHERE id = ?", // query must be normalized
		Func:	   "fetchUser", // optional
		File:	   "models/user.go", // optional
		Line:	   123, // optional
		StartTime: startTime,
		EndTime:   time.Now(),
	},
)
```

#### Sending queue stats

```go
metric := &gobrake.QueueMetric{
	Queue:   "my-queue-name",
	Errored: true,
}

ctx, span := metric.Start(ctx, "sql")
users, err := fetchUser(ctx, userID)
span.Finish()

ctx, span = metric.Start(ctx, "http")
resp, err := http.Get("http://example.com/")
span.Finish()

notifier.Queues.Notify(ctx, metric)
```

Additional notes
----------------

### Exception limit

The maximum size of an exception is 64KB. Exceptions that exceed this limit
will be truncated to fit the size.

### Logging

There's a [glog fork][glog], which integrates with Gobrake. It provides all of
original glog's functionality and adds the ability to send errors/logs to
[Airbrake.io][airbrake.io].

Supported Go versions
---------------------

The library supports Go v1.11+. The CI file would be the best source of truth
because it contains all Go versions that we test against.

Contact
-------

In case you have a problem, question or a bug report, feel free to:

* [file an issue][issues]
* [send us an email](mailto:support@airbrake.io)
* [tweet at us][twitter]
* chat with us (visit [airbrake.io][airbrake.io] and click on the round orange
	button in the bottom right corner)

License
-------

The project uses the MIT License. See LICENSE.md for details.

[arthur-go]: http://f.cl.ly/items/3J3h1L05222X3o1w2l2L/golang.jpg
[airbrake.io]: https://airbrake.io
[gobrake-github]: https://github.com/airbrake/gobrake
[docs]: https://pkg.go.dev/github.com/airbrake/gobrake/v5
[docs/performance]: https://airbrake.io/docs/performance-monitoring/go/
[beego]: https://beego.me
[gin]: https://github.com/gin-gonic/gin
[negroni]: https://github.com/urfave/negroni
[semver2]: http://semver.org/spec/v2.0.0.html
[go-mod]: https://github.com/golang/go/wiki/Modules
[project-idkey]: https://s3.amazonaws.com/airbrake-github-assets/gobrake/project-id-key.png
[issues]: https://github.com/airbrake/gobrake/issues
[twitter]: https://twitter.com/airbrake
[glog]: https://github.com/airbrake/glog
