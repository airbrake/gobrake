package gobrake

import (
	"fmt"
	"net/http"

	"github.com/golang/glog"
)

const (
	notifierVersion = "1.0"
	notifierURL     = "http://github.com/airbrake/gobrake"
)

var (
	_ Notifier = &StdNotifier{}
)

type Transporter interface {
	transport(error, []*stackEntry, *http.Request, map[string]string, map[string]interface{}) error
}

type Notifier interface {
	Transport() Transporter
	SetContext(string, string)
	Notify(error, *http.Request, map[string]interface{}) error
	NotifyPanic(interface{}, *http.Request, map[string]interface{}) error
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

func (n *StdNotifier) Notify(
	e error, r *http.Request, session map[string]interface{},
) error {
	stack := stack(1, n.StackFilter)

	context := make(map[string]string)
	for k, v := range n.context {
		context[k] = v
	}

	if err := n.t.transport(e, stack, r, context, session); err != nil {
		glog.Errorf("gobrake failed (%s) reporting error: %s", err, e)
		return err
	}

	return nil
}

func (n *StdNotifier) NotifyPanic(
	iface interface{}, r *http.Request, session map[string]interface{},
) error {
	switch v := iface.(type) {
	case error:
		return n.Notify(v, r, nil)
	case string:
		return n.Notify(newPanicStr(v), r, nil)
	default:
		s := fmt.Sprint(iface)
		return n.Notify(newPanicStr(s), r, nil)
	}
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
