Airbrake Golang Notifier
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

    var Notifier gobrake.Notifier

    func init() {
        transport := gobrake.NewJSONTransport(&http.Client{}, projectID, key, secure)
        Notifier = gobrake.NewNotifier(transport)
        Notifier.SetContext("environment", "production")
        Notifier.SetContext("version", "1.0")
    }

    func handler(w http.ResponseWriter, r *http.Request) {
        defer func() {
            if iface := recover(); iface != nil {
                Notifier.Panic(iface, r, nil)
            }
        }()

        if err := process(r); err != nil {
            go Notifier.Notify(err, r, nil)
        }
    }

Appengine
=========

    import (
        "net/http"

        "appengine/urlfetch"
        "github.com/airbrake/gobrake"
    )

    var (
        Notifier *ErrorNotifier
    )

    func init() {
        transport := gobrake.NewJSONTransport(
            nil, config.AirbrakeProjectID, config.AirbrakeKey, config.AirbrakeSecure)
        Notifier := gobrake.NewNotifier(transport)
        Notifier.SetContext("environment", config.Env)
        Notifier.SetContext("version", config.Version)

        Notifier = &ErrorNotifier{N: Notifier}
        http.Handle("/", &RecoverHandler{H: Router})
    }

    type ErrorNotifier struct {
        N gobrake.Notifier
    }

    func (n *ErrorNotifier) Notify(
        c appengine.Context, e error, r *http.Request, session map[string]interface{},
    ) {
        if v, ok := n.N.Transport().(*gobrake.JSONTransport); ok {
            v.Client = urlfetch.Client(c)
        }
        if err := n.N.Notify(e, r, session); err != nil {
            c.Errorf("Notify failed: %v", err)
        }
    }

    func (n *ErrorNotifier) Panic(
        c appengine.Context, e interface{}, r *http.Request, session map[string]interface{},
    ) {
        if v, ok := n.N.Transport().(*gobrake.JSONTransport); ok {
            v.Client = urlfetch.Client(c)
        }
        if err := n.N.Panic(e, r, session); err != nil {
            c.Errorf("Notify failed: %v", err)
        }
    }

    type RecoverHandler struct {
        H http.Handler
    }

    func (h *RecoverHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
        defer func() {
            c := appengine.NewContext(r)
            if iface := recover(); iface != nil {
                Notifier.Panic(c, iface, r, nil)
                panic(iface)
            }
        }()
        h.H.ServeHTTP(w, r)
    }
