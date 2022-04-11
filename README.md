<p align="center">
  <img src="https://airbrake-github-assets.s3.amazonaws.com/brand/airbrake-full-logo.png" width="200">
</p>

# Gobrake

[![.github/workflows/test.yml](https://github.com/airbrake/gobrake/actions/workflows/test.yml/badge.svg?branch=master)](https://github.com/airbrake/gobrake/actions/workflows/test.yml)
[![PkgGoDev](https://pkg.go.dev/badge/airbrake/gobrake)][docs]

* [Gobrake official documentation][docs-official]
* [pkg.go.dev documentation][docs]

## Introduction

_Gobrake_ is the official notifier package for [Airbrake][airbrake.io] for the
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
* Integrations with [Beego][beego], [Gin][gin], [Negroni][negroni] and [Fiber][fiber]
* Last but not least, we follow [semantic versioning 2.0.0][semver2]

## Installation

When using Go Modules, you do not need to install anything to start using Airbrake with your Go application. Import the package and the go tool will automatically download the latest version of the package when you next build your program.

```go
import (
  "github.com/airbrake/gobrake/v5"
)
```

With or without Go Modules, to use the latest version of the package, run:

```sh
go get github.com/airbrake/gobrake/v5
```

### Installing in a new project

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
    ProjectId:   <YOUR PROJECT ID>, // <-- Fill in this value
    ProjectKey:  "<YOUR API KEY>", // <-- Fill in this value
    Environment: "production",
})

func main() {
    defer airbrake.Close()

    airbrake.Notify(errors.New("operation failed"), nil)
}
```

### ProjectId & ProjectKey

You **must** set both `ProjectId` & `ProjectKey`.

To find your `ProjectId` (`int64`) and `ProjectKey` (`string`) navigate to your
project's _Settings_ and copy the values from the right sidebar.

![id-key][project-idkey]

## Getting Started

To check the complete Guide, visit our official [docs][docs-official].

## API

For complete API description please follow documentation on [pkg.go.dev
documentation][docs].

> **Note**: Gobrake provides middleware out of
the box and you may find our example apps more helpful:

* [Gin](examples/gin)
* [Fiber](examples/fiber)
* [Beego](examples/beego)
* [Negroni](examples/negroni)

## Additional notes

### Exception limit

The maximum size of an exception is 64KB. Exceptions that exceed this limit
will be truncated to fit the size.

### Logging

We support two major logging frameworks:

* There's a [glog fork][glog], which integrates with Gobrake. It provides all of
original glog's functionality and adds the ability to send errors/logs to
[Airbrake.io][airbrake.io].
* [apex/log][apexlog], to check how to integrate gobrake with apex/log, see [example](example/apexlog).

## Supported Go versions

The library supports Go v1.15+. The CI file would be the best source of truth
because it contains all Go versions that we test against.

## Contact

In case you have a problem, question or a bug report, feel free to:

* [file an issue][issues]
* [send us an email](mailto:support@airbrake.io)
* [tweet at us][twitter]
* chat with us (visit [airbrake.io][airbrake.io] and click on the round orange
    button in the bottom right corner)

## License

The project uses the MIT License. See [LICENSE.md](https://github.com/airbrake/gobrake/blob/master/LICENSE.md) for details.

[airbrake.io]: https://airbrake.io
[docs-official]: https://docs.airbrake.io/docs/platforms/go-lang/
[docs]: https://pkg.go.dev/github.com/airbrake/gobrake/v5
[docs/performance]: https://docs.airbrake.io/docs/overview/apm/#monitoring-go-apps
[beego]: https://github.com/beego/beego
[gin]: https://github.com/gin-gonic/gin
[negroni]: https://github.com/urfave/negroni
[fiber]: https://github.com/gofiber/fiber
[semver2]: http://semver.org/spec/v2.0.0.html
[go-mod]: https://github.com/golang/go/wiki/Modules
[project-idkey]: https://s3.amazonaws.com/airbrake-github-assets/gobrake/project-id-key.png
[issues]: https://github.com/airbrake/gobrake/issues
[twitter]: https://twitter.com/airbrake
[glog]: https://github.com/airbrake/glog
[apexlog]: https://github.com/apex/log
