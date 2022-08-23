package main

import (
	"fmt"
	"time"

	"github.com/airbrake/gobrake/v5"
	zapbrake "github.com/airbrake/gobrake/v5/zap"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var ProjectID int64 = 999999                               // Insert your Project ID here
var ProjectKey string = "xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx" // Insert your Project Key here

var notifier = gobrake.NewNotifierWithOptions(&gobrake.NotifierOptions{
	ProjectId:   ProjectID,
	ProjectKey:  ProjectKey,
	Environment: "production",
})

/*
Note: This example only shows how to send errors to Airbrake,
i.e., how to create a core and send the logs to Airbrake.
If you want to write the logs to a JSON file or stdout,
you should use `zapcore.NewTee()` for using multiple cores.
You can set the severity accordingly.
*/
func main() {
	zapbrake, err := zapbrake.NewCore(zapcore.ErrorLevel, notifier)
	if err != nil {
		// The only case when this error would be returned is when Airbrake was not set up and passed in as a nil value.
		// Either stop the execution of the code or ignore it, as zapbrake is set to an empty core that won't write to
		// Airbrake if a pointer to the Airbrake notifier is not provided.
		panic(err)
	}
	/*
		To use zapbrake core with other cores uncomment the below code and
		replace `zapbrake` with `core` in the logger
	*/
	// config := zap.NewProductionEncoderConfig()
	// config.EncodeTime = zapcore.ISO8601TimeEncoder
	// consoleEncoder := zapcore.NewConsoleEncoder(config)
	// core := zapcore.NewTee(
	// 	zapbrake,
	// 	zapcore.NewCore(consoleEncoder, zapcore.AddSync(os.Stdout), zapcore.ErrorLevel),
	// )
	logger := zap.New(zapbrake)
	// Define custom fields
	logger.With(
		zap.String("file", "something.png"),
		zap.String("type", "image/png"),
		zap.String("user", "tobi"),
	)
	logger.Info("upload")          // This log is not sent to Airbrake because the log level is set to Error
	logger.Info("upload complete") // This log is not sent to Airbrake because the log level is set to Error
	logger.Warn("upload retry")    // This log is not sent to Airbrake because the log level is set to Error

	logger.Error("upload failed") // This log is sent to Airbrake because the log level is set to Error
	logger.Sync()
	fmt.Printf("Check your Airbrake dashboard at https://YOUR_SUBDOMAIN.airbrake.io/projects/%v to see these error occurrences\n", ProjectID)
	time.Sleep(1 * time.Second)
}
