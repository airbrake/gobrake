package apexlog

import (
	"github.com/airbrake/gobrake/v5"

	"github.com/apex/log"
)

// Handler implementation.
type Handler struct {
	Gobrake         *gobrake.Notifier
	HandlerSeverity log.Level
}

// New returns a function that satisfies apex/log.Handler interface
func New(notifier *gobrake.Notifier, level log.Level) *Handler {
	h := Handler{notifier, level}
	return &h
}

// HandleLog method is used for sending notices to airbrake.
func (h *Handler) HandleLog(e *log.Entry) error {
	h.notifyAirbrake(e.Level, e.Message, e.Fields)
	return nil
}

func (h *Handler) notifyAirbrake(level log.Level, msg string, params log.Fields) {
	if level < h.HandlerSeverity {
		return
	}

	notice := gobrake.NewNotice(msg, nil, 0)
	notice.Context["severity"] = level.String()
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
