package zerolog

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/airbrake/gobrake/v5"
	"github.com/buger/jsonparser"
	"github.com/rs/zerolog"
)

type Writer struct {
	Gobrake *gobrake.Notifier
}

// New creates a new Writer
func New(notifier *gobrake.Notifier) *Writer {
	return &Writer{Gobrake: notifier}
}

// Write
func (w *Writer) Write(data []byte) (int, error) {
	lvlStr, err := jsonparser.GetUnsafeString(data, zerolog.LevelFieldName)
	if err != nil {
		return 0, fmt.Errorf("error getting zerolog level: %w", err)
	}

	lvl, err := zerolog.ParseLevel(lvlStr)
	if err != nil {
		return 0, fmt.Errorf("error parsing zerolog level: %w", err)
	}

	if lvl < zerolog.ErrorLevel || lvl > zerolog.PanicLevel {
		return len(data), nil
	}

	var userData interface{}
	err = json.Unmarshal(data, &userData)
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

	notice := gobrake.NewNotice(ze.message, nil, 6)
	notice.Context["severity"] = string(lvl)
	notice.Params["userData"] = userData
	notice.Error = errors.New(ze.error)
	w.Gobrake.SendNoticeAsync(notice)
	return len(data), nil
}

func (w *Writer) Close() error {
	w.Gobrake.Flush()
	return nil
}
