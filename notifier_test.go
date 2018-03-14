package gobrake_test

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"runtime"
	"testing"

	"github.com/airbrake/gobrake"
	"github.com/airbrake/gobrake/internal/testpkg1"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestGobrake(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "gobrake")
}

var _ = Describe("Notifier", func() {
	var notifier *gobrake.Notifier
	var sentNotice *gobrake.Notice
	var sendNoticeReq *http.Request

	notify := func(e interface{}, req *http.Request) {
		notifier.Notify(e, req)
		notifier.Flush()
	}

	BeforeEach(func() {
		handler := func(w http.ResponseWriter, req *http.Request) {
			sendNoticeReq = req

			b, err := ioutil.ReadAll(req.Body)
			if err != nil {
				panic(err)
			}

			sentNotice = new(gobrake.Notice)
			err = json.Unmarshal(b, sentNotice)
			Expect(err).To(BeNil())

			w.WriteHeader(http.StatusCreated)
			w.Write([]byte(`{"id":"123"}`))
		}
		server := httptest.NewServer(http.HandlerFunc(handler))

		notifier = gobrake.NewNotifier(1, "key")
		notifier.SetHost(server.URL)
	})

	AfterEach(func() {
		Expect(notifier.Close()).NotTo(HaveOccurred())
	})

	It("reports error and backtrace", func() {
		notify("hello", nil)

		e := sentNotice.Errors[0]
		Expect(e.Type).To(Equal("string"))
		Expect(e.Message).To(Equal("hello"))

		frame := e.Backtrace[0]
		Expect(frame.File).To(Equal("/GOPATH/github.com/airbrake/gobrake/notifier_test.go"))
		Expect(frame.Line).To(Equal(31))
		Expect(frame.Func).To(Equal("glob..func1.1"))
		Expect(frame.Code[31]).To(Equal("\t\tnotifier.Notify(e, req)"))
	})

	It("reports error and backtrace when error is created with pkg/errors", func() {
		err := testpkg1.Foo()
		notify(err, nil)
		e := sentNotice.Errors[0]

		Expect(e.Type).To(Equal("*errors.fundamental"))
		Expect(e.Message).To(Equal("Test"))

		frame := e.Backtrace[0]
		Expect(frame.File).To(Equal("/GOPATH/github.com/airbrake/gobrake/internal/testpkg1/testhelper.go"))
		Expect(frame.Line).To(Equal(10))
		Expect(frame.Func).To(Equal("Bar"))
		Expect(frame.Code[10]).To(Equal(`	return errors.New("Test")`))

		frame = e.Backtrace[1]
		Expect(frame.File).To(Equal("/GOPATH/github.com/airbrake/gobrake/internal/testpkg1/testhelper.go"))
		Expect(frame.Line).To(Equal(6))
		Expect(frame.Func).To(Equal("Foo"))
		Expect(frame.Code[6]).To(Equal("\treturn Bar()"))
	})

	It("reports context, env, session and params", func() {
		wanted := notifier.Notice("hello", nil, 3)
		wanted.Context["context1"] = "context1"
		wanted.Env["env1"] = "value1"
		wanted.Session["session1"] = "value1"
		wanted.Params["param1"] = "value1"

		id, err := notifier.SendNotice(wanted)
		Expect(err).To(BeNil())
		Expect(id).To(Equal("123"))

		Expect(sentNotice).To(Equal(wanted))
	})

	It("passes token by header 'Authorization: Bearer {project key}'", func() {
		Expect(sendNoticeReq.Header.Get("Authorization")).To(Equal("Bearer key"))
	})

	It("reports context using SetContext", func() {
		notifier.AddFilter(func(notice *gobrake.Notice) *gobrake.Notice {
			notice.Context["environment"] = "production"
			return notice
		})
		notify("hello", nil)

		Expect(sentNotice.Context["environment"]).To(Equal("production"))
	})

	It("reports request", func() {
		u, err := url.Parse("http://foo/bar")
		Expect(err).To(BeNil())

		req := &http.Request{
			Method: "GET",
			URL:    u,
			Header: http.Header{
				"User-Agent": {"my_user_agent"},
				"X-Real-Ip":  {"127.0.0.1"},
				"h1":         {"h1v1", "h1v2"},
				"h2":         {"h2v1"},
			},
			Form: url.Values{
				"f1": {"f1v1"},
				"f2": {"f2v1", "f2v2"},
			},
		}

		notify("hello", req)

		ctx := sentNotice.Context
		Expect(ctx["url"]).To(Equal("http://foo/bar"))
		Expect(ctx["httpMethod"]).To(Equal("GET"))
		Expect(ctx["userAgent"]).To(Equal("my_user_agent"))
		Expect(ctx["userAddr"]).To(Equal("127.0.0.1"))

		env := sentNotice.Env
		Expect(env["h1"]).To(Equal([]interface{}{"h1v1", "h1v2"}))
		Expect(env["h2"]).To(Equal("h2v1"))
	})

	It("collects and reports some context", func() {
		notify("hello", nil)

		hostname, _ := os.Hostname()
		gopath := os.Getenv("GOPATH")
		wd, _ := os.Getwd()

		Expect(sentNotice.Context["language"]).To(Equal(runtime.Version()))
		Expect(sentNotice.Context["os"]).To(Equal(runtime.GOOS))
		Expect(sentNotice.Context["architecture"]).To(Equal(runtime.GOARCH))
		Expect(sentNotice.Context["hostname"]).To(Equal(hostname))
		Expect(sentNotice.Context["rootDirectory"]).To(Equal(wd))
		Expect(sentNotice.Context["gopath"]).To(Equal(gopath))
		Expect(sentNotice.Context["component"]).To(Equal("github.com/airbrake/gobrake_test"))
		Expect(sentNotice.Context["revision"]).NotTo(BeEmpty())
	})

	It("does not panic on double close", func() {
		Expect(notifier.Close()).NotTo(HaveOccurred())
	})

	It("allows setting custom severity", func() {
		customSeverity := "critical"

		notice := notifier.Notice("hello", nil, 3)
		notice.Context["severity"] = customSeverity

		notify(notice, nil)
		Expect(sentNotice.Context["severity"]).To(Equal(customSeverity))
	})
})

var _ = Describe("rate limiting", func() {
	var notifier *gobrake.Notifier
	var requests int

	BeforeEach(func() {
		handler := func(w http.ResponseWriter, req *http.Request) {
			requests++
			w.Header().Set("X-RateLimit-Delay", "10")
			w.WriteHeader(429)
		}
		server := httptest.NewServer(http.HandlerFunc(handler))

		notifier = gobrake.NewNotifier(1, "key")
		notifier.SetHost(server.URL)
	})

	AfterEach(func() {
		Expect(notifier.Close()).NotTo(HaveOccurred())
	})

	It("pauses notifier", func() {
		notice := notifier.Notice("hello", nil, 3)
		for i := 0; i < 3; i++ {
			_, err := notifier.SendNotice(notice)
			Expect(err).To(MatchError("gobrake: IP is rate limited"))
		}
		Expect(requests).To(Equal(1))
	})
})
