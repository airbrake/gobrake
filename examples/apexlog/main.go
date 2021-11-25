package main

import (
	"errors"
	"fmt"

	"github.com/airbrake/gobrake/v5"
	"github.com/airbrake/gobrake/v5/apexlog"
	"github.com/apex/log"
)

var ProjectId int64 = 363389
var ProjectKey string = "baa755018d5e35e07897ee0087fcce9c"

func main() {
	airbrake := gobrake.NewNotifierWithOptions(&gobrake.NotifierOptions{
		ProjectId:   ProjectId,
		ProjectKey:  ProjectKey,
		Environment: "production",
	})

	defer airbrake.Close()

	// Create the airbrake handler for "github.com/apex/log"
	// Insert the airbrake notifier instance and set the severity level.
	// The acceptable severity levels are "DebugLog" "InfoLog", "WarnLog" and "ErrorLog".
	// If the severity level is not recognized, "error" severity is used.
	// Note: The severity is different from level used by log package to make sure that you can send logs to airbrake accordingly.
	airbrakeHandler := apexlog.New(airbrake, apexlog.ErrorLog)

	log.SetLevel(log.DebugLevel)

	// Set the airbrake handler in the log
	log.SetHandler(airbrakeHandler)

	ctx := log.WithFields(log.Fields{
		"file": "something.png",
		"type": "image/png",
		"user": "tobi",
	})

	fmt.Printf("Check your Airbrake dashboard at https://airbrake.io/projects/%v to see these log messages\n", ProjectId)

	ctx.Info("upload")
	ctx.Info("upload complete")
	ctx.Warn("upload retry")
	ctx.WithError(errors.New("unauthorized")).Error("upload failed")
	ctx.Errorf("failed to upload %s", "img.png")
}