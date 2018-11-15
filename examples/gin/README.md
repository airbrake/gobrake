# Gin integration

This is an example of basic Gin app with Airbrake middleware that reports route stats.

## How to run API

```bash
cd $GOPATH/src/github.com/airbrake/gobrake/examples/gin
go run *.go -env=production -airbrake_project_id=123456 -airbrake_project_key=FIXME
```

Visit http://127.0.0.1:8080/hello/{name}
