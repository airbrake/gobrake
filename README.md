Airbrake Golang notifier
========================

Example:

    import (
        "github.com/airbrake/gobrake"
    )

    var notifier = gobrake.NewNotifier(gobrake.Config{
        APIKey: "apikey",
        AppEnv: "production",
        AppVersion: "1.0",
    })

    func handler(w http.ResponseWriter, r *http.Request) {
        if err := process(r); err != nil {
            notifier.Notify(err, r)
        }
    }
