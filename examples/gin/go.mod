module github.com/airbrake/gobrake/v5/examples/gin

go 1.15

replace github.com/airbrake/gobrake/v5 => ../..

replace github.com/airbrake/gobrake/gin => ../../gin

require (
	github.com/airbrake/gobrake/gin v0.0.0-00010101000000-000000000000
	github.com/airbrake/gobrake/v5 v5.0.2
	github.com/gin-gonic/gin v1.6.3
)
