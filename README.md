# Airbrake Golang Notifier [![Build Status](https://travis-ci.org/airbrake/gobrake.svg?branch=v2)](https://travis-ci.org/airbrake/gobrake)

<img src="http://f.cl.ly/items/3J3h1L05222X3o1w2l2L/golang.jpg" width=800px>

# Example

```go
package main

import (
    "errors"

    "github.com/airbrake/gobrake"
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

```go
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

```go
notice := airbrake.NewNotice("operation failed", nil, 3)
notice.Context["severity"] = "critical"
airbrake.Notify(notice, nil)
```

## Logging

You can use [glog fork](https://github.com/airbrake/glog) to send your logs to Airbrake.

## Sending routes stats

In order to collect some basic routes stats you can instrument your application using `notifier.Routes.Notify` API:

```go
notifier.Routes.Notify(ctx, &gobrake.RouteTrace{
    Method:     c.Request.Method,
    Route:      routeName,
    StatusCode: c.Writer.Status(),
    Start:      startTime,
    End:        time.Now(),
})
```

We also prepared HTTP middlewares for [Gin](examples/gin) and [Beego](examples/beego) users.

To get more detailed timing you can wrap important blocks of code into spans. For example, you can create 2 spans `sql` and `http` to measure timing of specific operations:

``` go
trace := &gobrake.RouteTrace{
    Method: c.Request.Method,
    Route:  routeName,
    Start:  time.Now(),
}

trace.StartSpan("sql")
users, err := fetchUser(ctx, userID)
trace.EndSpan("sql")

trace.StartSpan("http")
resp, err := http.Get("http://example.com/")
trace.EndSpan("http")

trace.StatusCode = http.StatusOK
notifier.Routes.Notify(ctx, trace)
```

You can also collect stats about individual SQL queries performance using following API:

```go
notifier.Queries.Notify(&gobrake.QueryInfo{
    Query: "SELECT * FROM users WHERE id = ?", // query must be normalized
    Func:  "optional function name",
    File:  "optional file name",
    Line:  123,
    Start: startTime,
    End:   time.Now(),
})
```
