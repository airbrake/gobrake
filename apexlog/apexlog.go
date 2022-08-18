package apexlog

import (
	"errors"

	"github.com/airbrake/gobrake/v5"

	"github.com/apex/log"
)

// Handler implementation.
type Handler struct {
	Gobrake         *gobrake.Notifier
	HandlerSeverity log.Level
	depth           int
}

func NewLogger(h *Handler) (*Handler, error) {
	if h.Gobrake == nil {
		return h, errors.New("airbrake notifier not defined")
	}
	h = &Handler{h.Gobrake, h.HandlerSeverity, h.depth}
	return h, nil
}

// New returns a function that satisfies apex/log.Handler interface
func New(notifier *gobrake.Notifier, level log.Level) *Handler {
	h, _ := NewLogger(&Handler{
		Gobrake:         notifier,
		HandlerSeverity: level,
		depth:           4,
	})
	return h
}

// SetDepth method is for setting the depth of the notices
func (h *Handler) SetDepth(depth int) {
	h.depth = depth
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

	notice := gobrake.NewNotice(msg, nil, h.depth)
	parameters := asParams(params)
	for key, parameter := range parameters {
		if key == "httpMethod" || key == "route" {
			notice.Context[key] = parameter
			delete(parameters, key)
		}
	}
	notice.Context["severity"] = level.String()
	notice.Params = parameters

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
