package main

import (
	"flag"
	"log"
	"net/http"

	"github.com/airbrake/gobrake/v4"
	ng "github.com/airbrake/gobrake/v4/negroni"

	"github.com/gorilla/mux"
	"github.com/urfave/negroni"
)

var env = flag.String("env", "development", "environment, e.g. development or production")
var projectID = flag.Int64("airbrake_project_id", 0, "airbrake project ID")
var projectKey = flag.String("airbrake_project_key", "", "airbrake project key")

func main() {
	flag.Parse()
	n := negroni.Classic()
	notifier := gobrake.NewNotifierWithOptions(&gobrake.NotifierOptions{
		ProjectId:   *projectID,
		ProjectKey:  *projectKey,
		Environment: *env,
	})
	n.Use(ng.NewMiddleware(notifier))
	r := mux.NewRouter()
	r.HandleFunc("/ping", ping)
	n.UseHandler(r)
	log.Fatal(http.ListenAndServe(":8080", n))
}

func ping(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("Pong"))
}
