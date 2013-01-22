package gobrake

import (
	"errors"
	"fmt"
	"net/http"
	"testing"

	. "launchpad.net/gocheck"
)

// Hook up gocheck into the gotest runner.
func Test(t *testing.T) { TestingT(t) }

type NotifierTest struct {
	xmlTransport  *XMLTransport
	jsonTransport *JSONTransport

	xmlNotifier, jsonNotifier *StdNotifier

	err error
	req *http.Request
}

var _ = Suite(&NotifierTest{})

func (t *NotifierTest) SetUpTest(c *C) {
	t.xmlTransport = NewXMLTransport(&http.Client{}, "apikey", true)
	t.xmlTransport.CreateAPIURL = "http://localhost:8080/notifier_api/v2/notices"

	t.jsonTransport = NewJSONTransport(&http.Client{}, 1, "apikey", true)
	t.jsonTransport.CreateAPIURL = "http://localhost:8080/api/v3/projects/1/notices?key=apikey"

	t.xmlNotifier = NewNotifier(t.xmlTransport)
	t.xmlNotifier.SetContext("environment", "production")
	t.xmlNotifier.SetContext("version", "1.0")
	t.xmlNotifier.SetContext("rootDirectory", "/approot")

	t.jsonNotifier = NewNotifier(t.jsonTransport)
	t.jsonNotifier.SetContext("environment", "production")
	t.jsonNotifier.SetContext("version", "1.0")
	t.jsonNotifier.SetContext("rootDirectory", "/approot")

	var err error
	t.err = errors.New("unexpected error")
	t.req, err = http.NewRequest("GET", "http://airbrake.io/", nil)
	c.Assert(err, IsNil)
	t.req.RemoteAddr = "127.0.0.1"
}

func (t *NotifierTest) TestXMLNotify(c *C) {
	c.Assert(t.xmlNotifier.Notify(t.err, nil, nil), IsNil)
}

func (t *NotifierTest) TestXMLNotifyWithRequest(c *C) {
	c.Assert(t.xmlNotifier.Notify(t.err, t.req, nil), IsNil)
}

func (t *NotifierTest) TestJSONNotify(c *C) {
	c.Assert(t.jsonNotifier.Notify(t.err, nil, nil), IsNil)
}

func (t *NotifierTest) TestJSONNotifyWithRequest(c *C) {
	c.Assert(t.jsonNotifier.Notify(t.err, t.req, nil), IsNil)
}

func (t *NotifierTest) TestNilError(c *C) {
	t.xmlNotifier.Notify(nil, nil, nil)
}

func (t *NotifierTest) TestPanic(c *C) {
	defer func() {
		if iface := recover(); iface != nil {
			t.jsonNotifier.Panic(iface, nil, nil)
		}
	}()
	panic("hello")
}

func (t *NotifierTest) TestFmtErrorf(c *C) {
	err := fmt.Errorf("hello")
	t.jsonNotifier.Notify(err, nil, nil)
}

// func (t *NotifierTest) TestDeploy(c *C) {
// 	c.Assert(t.xmlNotifier.Deploy("", "", ""), IsNil)
// }
