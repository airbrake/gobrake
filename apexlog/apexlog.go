package apexlog

import (
	"fmt"

	"github.com/airbrake/gobrake/v5"

	"github.com/apex/log"
)

// severity identifies the sort of log: info, warn etc.
type severity int

// These constants identify the log levels in order of increasing severity.
// A message written to a high-severity log file is also written to each
// lower-severity log file.
const (
	DebugLog severity = iota
	InfoLog
	WarnLog
	ErrorLog
)

var severityName = []string{
	DebugLog: "debug",
	InfoLog:  "info",
	WarnLog:  "warn",
	ErrorLog: "error",
}

// String implementation for severity.
func (s severity) String() string {
	return severityName[s]
}

// Handler implementation.
type Handler struct {
	Gobrake         *gobrake.Notifier
	HandlerSeverity severity
}

// New Apex Logs handler for airbrake.
func New(notifier *gobrake.Notifier, s severity) *Handler {
	h := Handler{notifier, s}
	return &h
}

// HandleLog method is used for sending notices to airbrake.
func (h *Handler) HandleLog(e *log.Entry) error {
	h.notifyAirbrake(severity(e.Level), e.Message, e.Fields)
	return nil
}

func (h *Handler) notifyAirbrake(s severity, arg interface{}, params log.Fields) {
	if s < h.HandlerSeverity {
		return
	}

	msg := fmt.Sprint(arg)
	notice := gobrake.NewNotice(msg, nil, 0)
	notice.Context["severity"] = s.String()
	notice.Params = asParams(params)

	h.Gobrake.Notify(notice, nil)
}

func asParams(data log.Fields) map[string]interface{} {
	params := make(map[string]interface{}, len(data))
	for k, v := range data {
		switch v := v.(type) {
		case error:
			params[k] = v.Error()
		default:
			params[k] = v
		}
	}
	return params
}
