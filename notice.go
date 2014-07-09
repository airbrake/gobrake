package gobrake

import (
	"fmt"
	"net/http"
)

func getCreateNoticeURL(projectId int64, key string) string {
	return fmt.Sprintf(
		"https://airbrake.io/api/v3/projects/%d/notices?key=%s",
		projectId, key,
	)
}

type Error struct {
	Type      string        `json:"type"`
	Message   string        `json:"message"`
	Backtrace []*StackEntry `json:"backtrace"`
}

type Notice struct {
	Notifier struct {
		Name    string `json:"name"`
		Version string `json:"version"`
		URL     string `json:"url"`
	} `json:"notifier"`
	Errors  []*Error               `json:"errors"`
	Context map[string]string      `json:"context"`
	Env     map[string]interface{} `json:"environment"`
	Session map[string]interface{} `json:"session"`
	Params  map[string]interface{} `json:"params"`
}

func NewNotice(
	e interface{},
	stack []*StackEntry,
	req *http.Request,
) *Notice {
	notice := &Notice{
		Errors: []*Error{
			&Error{
				Type:      fmt.Sprintf("%T", e),
				Message:   fmt.Sprint(e),
				Backtrace: stack,
			},
		},
		Context: make(map[string]string),
		Env:     make(map[string]interface{}),
		Session: make(map[string]interface{}),
		Params:  make(map[string]interface{}),
	}

	notifier := &notice.Notifier
	notifier.Name = "gobrake"
	notifier.Version = "1.0"
	notifier.URL = "https://github.com/airbrake/gobrake"

	if req != nil {
		notice.Context["url"] = req.URL.String()
		if ua := req.Header.Get("User-Agent"); ua != "" {
			notice.Context["userAgent"] = ua
		}

		for k, v := range req.Header {
			if len(v) == 1 {
				notice.Env[k] = v[0]
			} else {
				notice.Env[k] = v
			}
		}

		if err := req.ParseForm(); err == nil {
			for k, v := range req.Form {
				if len(v) == 1 {
					notice.Params[k] = v[0]
				} else {
					notice.Params[k] = v
				}
			}
		}
	}

	return notice
}
