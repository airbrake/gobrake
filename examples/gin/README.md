# Gin API example

An example of using the Gin framework to create middleware that reports airbrake route stats.

## How to run API
```bash
$ cd $GOPATH/src/github.com/airbrake/gobrake/examples/gin
$ go run *.go -env=production -airbrake_project_id=123456 -airbrake_project_key=FIXME -airbrake_host=https://getexceptional.airbrake.io
```

Go to http://127.0.0.1:8080/hello/{name}