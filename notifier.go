package gobrake

import (
	"fmt"
	"log"
	"net/http"
	"os"
)

const (
	notifierVersion = "1.0"
	notifierURL     = "http://github.com/airbrake/gobrake"
)

var (
	Logger = log.New(os.Stderr, "", log.LstdFlags)

	_ Notifier = &StdNotifier{}
)

type Transporter interface {
	Transport(error, []*stackEntry, *http.Request, map[string]string, map[string]interface{}) error
}

type Notifier interface {
	Transport() Transporter
	SetContext(string, string)
	Notify(error, *http.Request, map[string]interface{}) error
	Panic(interface{}, *http.Request, map[string]interface{}) error
	Deploy(string, string, string) error
}

type StdNotifier struct {
	StackFilter func(string, int, string, string) bool

	t       Transporter
	context map[string]string
}

func NewNotifier(t Transporter) *StdNotifier {
	return &StdNotifier{
		StackFilter: stackFilter,

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

	stack := stack(1, n.StackFilter)

	context := make(map[string]string)
	for k, v := range n.context {
		context[k] = v
	}

	if err := n.t.Transport(e, stack, r, context, session); err != nil {
		Logger.Printf("gobrake: Transport failed: %v", err)
		return err
	}

	return nil
}

func (n *StdNotifier) Panic(
	iface interface{}, r *http.Request, session map[string]interface{},
) error {
	switch v := iface.(type) {
	case error:
		return n.Notify(v, r, nil)
	case string:
		return n.Notify(newPanicStr(v), r, nil)
	}
	s := fmt.Sprint(iface)
	return n.Notify(newPanicStr(s), r, nil)
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

//------------------------------------------------------------------------------

func newPanicStr(s string) error {
	return &panicStr{s}
}

type panicStr struct {
	s string
}

func (e *panicStr) Error() string {
	return e.s
}

func (n *panicStr) RuntimeError() {}
