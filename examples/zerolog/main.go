package main

import (
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/airbrake/gobrake/v5"
	airbrakeWriter "github.com/airbrake/gobrake/v5/zerolog"
	"github.com/rs/zerolog"
)

var ProjectId int64 = 999999                               // Insert your Project Id here
var ProjectKey string = "xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx" // Insert your Project Key here

func main() {
	airbrake := gobrake.NewNotifierWithOptions(&gobrake.NotifierOptions{
		ProjectId:   ProjectId,
		ProjectKey:  ProjectKey,
		Environment: "production",
	})

	defer airbrake.Close()

	// Note: This writer only accepts errors, fatal and panic logs, all others will be ignored.
	// You can still send logs to stdout or another writer via io.MultiWriter
	w := airbrakeWriter.New(airbrake)

	// Insert the newly created airbrakeWriter into zerolog
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

	fmt.Printf("Check your Airbrake dashboard at https://airbrake.io/projects/%v to see these log messages\n", ProjectId)

	loggerWithData.Info().Msg("upload")       // will `not` be sent to airbrake but sent to stdout
	loggerWithData.Warn().Msg("upload retry") // will `not` be sent to airbrake but sent to stdout
	// This error will be sent both to stdout and to airbrake via the MultiWriter pipeline
	loggerWithData.Error().Err(errors.New("unauthorized")).Msg("upload failed") // will `be` sent to airbrake & stdout
}
