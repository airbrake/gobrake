package zerolog

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"

	"github.com/airbrake/gobrake/v5"
	"github.com/buger/jsonparser"
	"github.com/rs/zerolog"
)

type WriteCloser struct {
	Gobrake *gobrake.Notifier
}

// Validates the WriteCloser matches the io.WriteCloser interface
var _ io.WriteCloser = (*WriteCloser)(nil)

// New creates a new WriteCloser
func New(notifier *gobrake.Notifier) (io.WriteCloser, error) {
	if notifier == nil {
		return &WriteCloser{}, errors.New("airbrake notifier not provided")
	}
	return &WriteCloser{Gobrake: notifier}, nil
}

// Write parses the log data and sends off error notices to airbrake
func (w *WriteCloser) Write(data []byte) (int, error) {
	lvl, err := jsonparser.GetUnsafeString(data, zerolog.LevelFieldName)
	if err != nil {
		return 0, fmt.Errorf("error getting zerolog level: %w", err)
	}

	if lvl != zerolog.ErrorLevel.String() {
		return len(data), nil
	}

	var logEntryData interface{}
	err = json.Unmarshal(data, &logEntryData)
	if err != nil {
		return 0, fmt.Errorf("error unmarshalling logs: %w", err)
	}
	type zeroError struct {
		message string
		error   string
	}
	var ze zeroError
	_ = jsonparser.ObjectEach(data, func(key, value []byte, vt jsonparser.ValueType, offset int) error {
		switch string(key) {
		case zerolog.MessageFieldName:
			ze.message = string(value)
		case zerolog.ErrorFieldName:
			ze.error = string(value)
		}

		return nil
	})

	// If gobrake was not setup but the writer was still used, ignore gobrake.
	if w.Gobrake == nil {
		return len(data), nil
	}

	notice := gobrake.NewNotice(ze.message, nil, 6)
	notice.Context["severity"] = lvl

	// Check for the following 2 fields in logEntryData to see if they
	// can be moved to the `Notice.Context`. Doing so would automatically link
	// them in airbrake.io dashboards.
	if asMap, ok := logEntryData.(map[string]interface{}); ok {
		const HttpMethod = "httpMethod"
		const Route = "route"

		if method, ok := asMap[HttpMethod].(string); ok {
			notice.Context[HttpMethod] = method
			delete(asMap, HttpMethod)
		}

		if route, ok := asMap[Route].(string); ok {
			notice.Context[Route] = route
			delete(asMap, Route)
		}
	}

	notice.Params["logEntryData"] = logEntryData
	notice.Error = errors.New(ze.error)
	w.Gobrake.SendNoticeAsync(notice)
	return len(data), nil
}

// Close flushes any remaining notices left in gobrake queue
func (w *WriteCloser) Close() error {
	w.Gobrake.Flush()
	return nil
}
