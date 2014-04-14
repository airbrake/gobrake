package gobrake

import (
	"errors"
	"fmt"
	"testing"
	"net/http"
	"net/http/httptest"

	. "gopkg.in/check.v1"
)

func Test(t *testing.T) { TestingT(t) }

type NotifierTest struct {
	xmlTransport  *XMLTransport
	jsonTransport *JSONTransport

	xmlNotifier, jsonNotifier *StdNotifier

	err error
	req *http.Request
	fakeServer  *httptest.Server
}

var _ = Suite(&NotifierTest{})

func (t *NotifierTest) SetUpTest(c *C) {

	t.fakeServer = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Content-Type") == "application/json" {
			w.WriteHeader(http.StatusCreated)
		} else {
			w.WriteHeader(http.StatusOK)
		}
		fmt.Sprintf("%v", 1)
	}))

	t.xmlTransport = NewXMLTransport("apikey", true)
	t.xmlTransport.CreateAPIURL = t.fakeServer.URL

	t.jsonTransport = NewJSONTransport(1, "apikey", true)
	t.jsonTransport.CreateAPIURL = t.fakeServer.URL

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

func (t *NotifierTest) TearDownTest(c *C) {
	t.fakeServer.Close()
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
	e := errors.New("")
	t.xmlNotifier.Notify(e, nil, nil)
}

func (t *NotifierTest) TestPanic(c *C) {
	defer func() {
		if iface := recover(); iface != nil {
			t.jsonNotifier.NotifyPanic(iface, nil, nil)
		}
	}()
	panic("hello")
}

func (t *NotifierTest) TestFmtErrorf(c *C) {
	err := fmt.Errorf("hello")
	t.jsonNotifier.Notify(err, nil, nil)
}
