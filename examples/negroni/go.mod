module github.com/airbrake/gobrake/v4/examples/negroni

go 1.15

replace github.com/airbrake/gobrake/v4/negroni => ../../negroni

require (
	github.com/airbrake/gobrake/v4 v4.2.0
	github.com/airbrake/gobrake/v4/negroni v0.0.0-00010101000000-000000000000 // indirect
	github.com/gorilla/mux v1.8.0
	github.com/urfave/negroni v1.0.0
)
