package gobrake

import (
	"log"
	"net/http"
	"os"
)

const (
	notifierVersion = "1.0"
	notifierURL     = "http://github.com/airbrake/gobrake"
)

var (
	deployURL       = "://go-airbrake.appspot.com/deploys.txt"
	createNoticeURL = "://go-airbrake.appspot.com/notifier_api/v2/notices"

	Logger = log.New(os.Stderr, "", log.LstdFlags)

	_ Notifier = &StdNotifier{}
)

type Transporter interface {
	Transport(error, *http.Request, map[string]string, map[string]interface{}) error
}

type Notifier interface {
	Transport() Transporter
	SetContext(string, string)
	Notify(error, *http.Request, map[string]interface{}) error
	Deploy(string, string, string) error
}

type StdNotifier struct {
	t       Transporter
	context map[string]string
}

func NewNotifier(t Transporter) *StdNotifier {
	return &StdNotifier{
		t:       t,
		context: make(map[string]string),
	}
}

func (n *StdNotifier) Transport() Transporter {
	return n.t
}

func (n *StdNotifier) SetContext(name, value string) {
	n.context[name] = value
}

func (n *StdNotifier) Notify(e error, r *http.Request, session map[string]interface{}) error {
	if e == nil {
		Logger.Printf("gobrake: error is nil")
		return nil
	}

	context := make(map[string]string)
	for k, v := range n.context {
		context[k] = v
	}

	if err := n.t.Transport(e, r, context, session); err != nil {
		Logger.Printf("gobrake: Transport failed: %v", err)
		return err
	}

	return nil
}

func (n *StdNotifier) Deploy(repository, revision, username string) error {
	return nil
	// req, err := http.NewRequest("POST", "", nil)
	// if err != nil {
	// 	return err
	// }
	// req.Form = url.Values{
	// 	"api_key":                {n.APIKey()},
	// 	"deploy[rails_env]":      {n.EnvName()},
	// 	"deploy[scm_repository]": {repository},
	// 	"deploy[scm_revision]":   {revision},
	// 	"deploy[local_username]": {username},
	// }

	// resp, err := http.DefaultClient.Do(req)
	// if err != nil {
	// 	return err
	// }
	// defer resp.Body.Close()
	// if code := resp.StatusCode; code != http.StatusOK {
	// 	return fmt.Errorf("gobrake: got %v response, expected 200 OK", code)
	// }

	// return nil
}
