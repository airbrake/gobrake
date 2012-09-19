Airbrake Golang notifier
========================

Example:

    import (
        "github.com/airbrake/gobrake"
    )

    var notifier = gobrake.NewNotifier("apikey", "production", "1.0", "")

    func handler(w http.ResponseWriter, r *http.Request) {
        if err := process(r); err != nil {
            notifier.Notify(err, r)
        }
    }
