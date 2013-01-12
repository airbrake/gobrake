Airbrake Golang notifier
========================

Example:

    import (
        "net/http"

        "github.com/airbrake/gobrake"
    )

    const (
        projectID = 1
        key = "apikey"
        secure = true
    )

    var notifier gobrake.Notifier

    func init() {
        transport := gobrake.NewJSONTransport(&http.Client{}, projectID, key, secure)
        notifier = gobrake.NewNotifier(transport)
        notifier.SetContext("environment", "production")
        notifier.SetContext("version", "1.0")
    }

    func handler(w http.ResponseWriter, r *http.Request) {
        if err := process(r); err != nil {
            go notifier.Notify(err, r, nil)
        }
    }
