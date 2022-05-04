package main

import (
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/airbrake/gobrake/v5"
	zerobrake "github.com/airbrake/gobrake/v5/zerolog" // Named import so that we don't conflict with rs/zerolog
	"github.com/rs/zerolog"
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

	// Note: This writer only accepts errors logs, all others will be ignored.
	// You can still send logs to stdout or another writer via io.MultiWriter
	w, err := zerobrake.New(airbrake)
	if err != nil {
		// The only way this error would be returned is if airbrake was not set up and passed in as a nil value
		// Either stop the execution of the code or ignore it, as w is set to an empty writer that wont write to
		// airbrake if a pointer to airbrake notifier is not provided.
		panic("airbrake was not setup correctly")
	}

	// Insert the newly created writer (w) into zerolog
	log := zerolog.New(io.MultiWriter(os.Stdout, w))
	zerolog.SetGlobalLevel(zerolog.DebugLevel)

	// Creates a sub logger that has additional ctx
	loggerWithData := log.With().
		Dict("ctx",
			zerolog.Dict().
				Str("BUILD", "gitsha").
				Str("VERSION", "").
				Str("TIME", ""),
		).
		Str("file", "something.png").
		Str("type", "image/png").
		Str("user", "tobi").
		Logger()

	fmt.Printf("Check your Airbrake dashboard at https://YOUR_SUBDOMAIN.airbrake.io/projects/%v to see these error occurrences\n", ProjectID)

	loggerWithData.Info().Msg("upload")       // Sends only to stdout because log level is Info
	loggerWithData.Warn().Msg("upload retry") // Sends only to stdout because log level is Warn.

	// This error is sent to Airbrake and stdout because the log level is Error
	loggerWithData.Error().Err(errors.New("unauthorized")).Msg("upload failed")
}
