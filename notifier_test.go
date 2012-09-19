package gobrake

import (
	"errors"
	"net/http"
	"testing"

	. "launchpad.net/gocheck"
)

func init() {
	deployURL = "//localhost:8080/deploys.txt"
	notifyURL = "//localhost:8080/notifier_api/v2/notices/"
}

// Hook up gocheck into the gotest runner.
func Test(t *testing.T) { TestingT(t) }

type NotifierTest struct {
	notifier Notifier
	err      error
}

var _ = Suite(&NotifierTest{})

func (t *NotifierTest) SetUpTest(c *C) {
	t.notifier = NewNotifier("apikey", "production", "1.0", "/approot")
	t.err = errors.New("unexpected error")
}

func (t *NotifierTest) TestNotifyWithoutRequest(c *C) {
	c.Assert(t.notifier.Notify(t.err, nil), IsNil)

	transport := &XMLTransport{}
	b, err := transport.marshal(t.notifier, t.err, nil)
	c.Assert(err, IsNil)
	c.Assert(string(b), Equals, expectedXML)
}

func (t *NotifierTest) TestNotifyWithRequest(c *C) {
	r, err := http.NewRequest("GET", "http://airbrake.io/", nil)
	c.Assert(err, IsNil)
	r.RemoteAddr = "127.0.0.1"

	c.Assert(t.notifier.Notify(t.err, r), IsNil)

	transport := &XMLTransport{}
	b, err := transport.marshal(t.notifier, t.err, r)
	c.Assert(err, IsNil)
	c.Assert(string(b), Equals, expectedXMLWithRequest)
}

func (t *NotifierTest) TestPanicWithNilError(c *C) {
	t.notifier.Notify(nil, nil)
}

func (t *NotifierTest) TestDeploy(c *C) {
	c.Assert(t.notifier.Deploy("", "", ""), IsNil)
}

var expectedXML = `<notice version="2.0"><api-key>apikey</api-key><notifier><name>Airbrake Go notifier</name><version>1.0</version><url>http://github.com/airbrake/goairbrake</url></notifier><error><class>*errors.errorString</class><message>unexpected error</message><backtrace><line file="/home/vmihailenco/workspace/go/src/pkg/reflect/value.go" number="521" method="reflect.Value.call"></line><line file="/home/vmihailenco/workspace/go/src/pkg/reflect/value.go" number="334" method="reflect.Value.Call"></line><line file="/home/vmihailenco/workspace/gocode/src/launchpad.net/gocheck/gocheck.go" number="709" method="launchpad.net/gocheck._func_006"></line><line file="/home/vmihailenco/workspace/gocode/src/launchpad.net/gocheck/gocheck.go" number="608" method="launchpad.net/gocheck._func_004"></line><line file="/home/vmihailenco/workspace/go/src/pkg/runtime/proc.c" number="271" method="runtime.goexit"></line></backtrace></error><request><url></url><component></component><action></action><cgi-data></cgi-data></request><server-environment><environment-name>production</environment-name><project-root>/approot</project-root><app-version>1.0</app-version></server-environment></notice>`

var expectedXMLWithRequest = `<notice version="2.0"><api-key>apikey</api-key><notifier><name>Airbrake Go notifier</name><version>1.0</version><url>http://github.com/airbrake/goairbrake</url></notifier><error><class>*errors.errorString</class><message>unexpected error</message><backtrace><line file="/home/vmihailenco/workspace/go/src/pkg/reflect/value.go" number="521" method="reflect.Value.call"></line><line file="/home/vmihailenco/workspace/go/src/pkg/reflect/value.go" number="334" method="reflect.Value.Call"></line><line file="/home/vmihailenco/workspace/gocode/src/launchpad.net/gocheck/gocheck.go" number="709" method="launchpad.net/gocheck._func_006"></line><line file="/home/vmihailenco/workspace/gocode/src/launchpad.net/gocheck/gocheck.go" number="608" method="launchpad.net/gocheck._func_004"></line><line file="/home/vmihailenco/workspace/go/src/pkg/runtime/proc.c" number="271" method="runtime.goexit"></line></backtrace></error><request><url>http://airbrake.io/</url><component></component><action></action><cgi-data><var key="METHOD">GET</var><var key="REMOTE_ADDR">127.0.0.1</var></cgi-data></request><server-environment><environment-name>production</environment-name><project-root>/approot</project-root><app-version>1.0</app-version></server-environment></notice>`
