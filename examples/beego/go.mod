module github.com/airbrake/gobrake/v5/examples/beego

go 1.15

replace github.com/airbrake/gobrake/v5 => ../..

replace github.com/airbrake/gobrake/v5/beego => ../../beego

require (
	github.com/airbrake/gobrake/v5 v5.0.0-00010101000000-000000000000
	github.com/airbrake/gobrake/v5/beego v0.0.0-00010101000000-000000000000
	github.com/astaxie/beego v1.12.2
)
