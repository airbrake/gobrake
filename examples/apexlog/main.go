package main

import (
	"errors"
	"fmt"

	"github.com/airbrake/gobrake/v5"
	"github.com/airbrake/gobrake/v5/apexlog"
	"github.com/apex/log"
)

var ProjectID int64 = 999999                               // Insert your Project ID here
var ProjectKey string = "xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx" // Insert your Project Key here

func main() {
	airbrake := gobrake.NewNotifierWithOptions(&gobrake.NotifierOptions{
		ProjectId:   ProjectID,
		ProjectKey:  ProjectKey,
		Environment: "production",
	})

	defer airbrake.Close()

	// Create the airbrake handler for "github.com/apex/log"
	// Insert the airbrake notifier instance and set the severity level.
	// The acceptable severity levels are "DebugLevel" "InfoLevel", "WarnLevel" and "ErrorLevel".
	// Note: The severity is different from level used by log package to make sure that you can send logs to airbrake accordingly.
	airbrakeHandler := apexlog.New(airbrake, log.ErrorLevel)

	log.SetLevel(log.DebugLevel)

	// Set the airbrake handler in the log
	log.SetHandler(airbrakeHandler)

	ctx := log.WithFields(log.Fields{
		"file": "something.png",
		"type": "image/png",
		"user": "tobi",
	})

	fmt.Printf("Check your Airbrake dashboard at https://YOUR_SUBDOMAIN.airbrake.io/projects/%v to see these error occurrences\n", ProjectID)

	ctx.Info("upload")
	ctx.Info("upload complete")
	ctx.Warn("upload retry")
	ctx.WithError(errors.New("unauthorized")).Error("upload failed")
	ctx.Errorf("failed to upload %s", "img.png")
}
