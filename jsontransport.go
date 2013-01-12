package gobrake

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
	"strconv"
	"strings"
)

var (
	createNoticeAPIV3URL = "//collect.airbrake.io/api/v3/projects/[PROJECT_ID]/notices?key=[KEY]"
)

type JSONTransport struct {
	CreateAPIURL string
	Client       *http.Client
}

func NewJSONTransport(
	client *http.Client, projectID int64, key string, isSecure bool,
) *JSONTransport {
	url := scheme(isSecure) + createNoticeAPIV3URL
	url = strings.Replace(url, "[PROJECT_ID]", strconv.FormatInt(projectID, 10), 1)
	url = strings.Replace(url, "[KEY]", key, 1)
	return &JSONTransport{
		CreateAPIURL: url,
		Client:       client,
	}
}

func (t *JSONTransport) Transport(
	e error, r *http.Request, context map[string]string, session map[string]interface{},
) error {
	jsonn := newJSONNotice(e, r, context, session)

	buf := &bytes.Buffer{}
	enc := json.NewEncoder(buf)
	if err := enc.Encode(jsonn); err != nil {
		return err
	}

	resp, err := t.Client.Post(t.CreateAPIURL, "application/json", buf)
	if err != nil {
		return err
	}
	resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return fmt.Errorf(
			"gobrake: got %v response, expected 201 CREATED", resp.StatusCode)
	}

	return nil
}

type jsonNotifier struct {
	Name    string `json:"name"`
	Version string `json:"version"`
	URL     string `json:"url"`
}

type jsonError struct {
	Type      string        `json:"type"`
	Message   string        `json:"message"`
	Backtrace []*stackEntry `json:"backtrace"`
}

type jsonNotice struct {
	Notifier *jsonNotifier          `json:"notifier"`
	Errors   []*jsonError           `json:"errors"`
	Context  map[string]string      `json:"context"`
	Env      map[string]interface{} `json:"environment"`
	Session  map[string]interface{} `json:"session"`
	Params   map[string]interface{} `json:"params"`
}

func newJSONNotice(
	e error, r *http.Request, context map[string]string, session map[string]interface{},
) *jsonNotice {
	notice := &jsonNotice{
		Notifier: &jsonNotifier{
			Name:    "Airbrake GO JSON Notifier",
			Version: notifierVersion,
			URL:     notifierURL,
		},
		Errors: []*jsonError{
			&jsonError{
				Type:      reflect.TypeOf(e).String(),
				Message:   e.Error(),
				Backtrace: stack(4),
			},
		},
		Context: context,
		Session: session,
	}

	if r != nil {
		context["url"] = r.URL.String()
		if ua := r.Header.Get("User-Agent"); ua != "" {
			context["browser"] = ua
		}
		notice.Context = context

		env := make(map[string]interface{}, len(r.Header))
		for k, _ := range r.Header {
			env[k] = r.Header.Get(k)
		}
		notice.Env = env

		if err := r.ParseForm(); err == nil {
			params := make(map[string]interface{}, len(r.Form))
			for k, _ := range r.Form {
				params[k] = r.Form.Get(k)
			}
			notice.Params = params
		}
	}

	return notice
}
