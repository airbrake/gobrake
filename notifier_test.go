package gobrake_test

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"regexp"
	"runtime"
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/airbrake/gobrake/v4"
	"github.com/airbrake/gobrake/v4/internal/testpkg1"
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

	var opt *gobrake.NotifierOptions

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
			_, err = w.Write([]byte(`{"id":"123"}`))
			Expect(err).To(BeNil())
		}
		server := httptest.NewServer(http.HandlerFunc(handler))

		opt = &gobrake.NotifierOptions{
			ProjectId:  1,
			ProjectKey: "key",
			Host:       server.URL,
		}
	})

	JustBeforeEach(func() {
		notifier = gobrake.NewNotifierWithOptions(opt)
	})

	AfterEach(func() {
		Expect(notifier.Close()).NotTo(HaveOccurred())
	})

	It("applies block list keys filter", func() {
		filter := gobrake.NewBlocklistKeysFilter("password", regexp.MustCompile("(?i)(user)"))
		notifier.AddFilter(filter)

		notice := &gobrake.Notice{
			Errors: []gobrake.Error{{
				Type:    "type1",
				Message: "msg1",
			}},
			Env: map[string]interface{}{
				"password": "slds2&LP",
				"User":     "username",
				"email":    "john@example.com",
			},
		}
		notifier.Notify(notice, nil)
		notifier.Flush()

		e := sentNotice.Errors[0]
		Expect(e.Type).To(Equal("type1"))
		Expect(e.Message).To(Equal("msg1"))
		Expect(sentNotice.Env).To(Equal(map[string]interface{}{
			"User":     "[Filtered]",
			"email":    "john@example.com",
			"password": "[Filtered]",
		}))
	})

	It("reports error and backtrace", func() {
		notify("hello", nil)

		e := sentNotice.Errors[0]
		Expect(e.Type).To(Equal("string"))
		Expect(e.Message).To(Equal("hello"))
		Expect(len(e.Backtrace)).NotTo(BeZero())
	})

	Context("DisableCodeHunks", func() {
		BeforeEach(func() {
			opt.DisableCodeHunks = true
		})

		AfterEach(func() {
			opt.DisableCodeHunks = false
		})

		It("does not report code hunks", func() {
			notify("hello", nil)

			for _, frame := range sentNotice.Errors[0].Backtrace {
				Expect(frame.Code).To(BeNil())
			}
		})
	})

	It("reports error and backtrace when error is created with pkg/errors", func() {
		err := testpkg1.Foo()
		notify(err, nil)
		e := sentNotice.Errors[0]

		Expect(e.Type).To(Equal("*errors.fundamental"))
		Expect(e.Message).To(Equal("Test"))

		frame := e.Backtrace[0]
		Expect(frame.File).To(ContainSubstring("gobrake/internal/testpkg1/testhelper.go"))
		Expect(frame.Line).To(Equal(10))
		Expect(frame.Func).To(Equal("Bar"))
		Expect(frame.Code[10]).To(Equal(`	return errors.New("Test")`))

		frame = e.Backtrace[1]
		Expect(frame.File).To(ContainSubstring("gobrake/internal/testpkg1/testhelper.go"))
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

		Expect(sentNotice.Context["context1"]).To(Equal(wanted.Context["context1"]))
		Expect(sentNotice.Env).To(Equal(wanted.Env))
		Expect(sentNotice.Session).To(Equal(wanted.Session))
		Expect(sentNotice.Params).To(Equal(wanted.Params))
	})

	It("sets context.severity=critical when notify on panic", func() {
		assert := func() {
			v := recover()
			Expect(v).NotTo(BeNil())

			e := sentNotice.Errors[0]
			Expect(e.Type).To(Equal("string"))
			Expect(e.Message).To(Equal("hello"))
			Expect(sentNotice.Context["severity"]).To(Equal("critical"))
		}

		defer assert()
		defer notifier.NotifyOnPanic()

		panic("hello")
	})

	It("passes token by header 'Authorization: Bearer {project key}'", func() {
		Expect(sendNoticeReq.Header.Get("Authorization")).To(Equal("Bearer key"))
	})

	It("sets user agent", func() {
		Expect(sendNoticeReq.Header.Get("User-Agent")).To(ContainSubstring("gobrake/"))
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
		Expect(sentNotice.Context["component"]).To(Equal("github.com/airbrake/gobrake/v4_test"))
		Expect(sentNotice.Context["repository"]).To(ContainSubstring("airbrake/gobrake"))
		Expect(sentNotice.Context["revision"]).NotTo(BeEmpty())
		Expect(sentNotice.Context["lastCheckout"]).NotTo(BeEmpty())
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

	It("filters errors with message that starts with '(string)Unsolicited response received on idle HTTP channel starting with", func() {
		sentNotice = nil

		msg := "Unsolicited response received on idle HTTP channel starting with HTTP/1.0 408 Request Time-out"
		notify(msg, nil)

		Expect(sentNotice).To(BeNil())
	})

	It("logs errors on invalid project id or key", func() {
		origLogger := gobrake.GetLogger()
		defer func() {
			gobrake.SetLogger(origLogger)
		}()

		buf := new(bytes.Buffer)
		l := log.New(buf, "", 0)
		gobrake.SetLogger(l)

		n := gobrake.NewNotifierWithOptions(
			&gobrake.NotifierOptions{
				ProjectId:  1,
				ProjectKey: "broken-key",
			},
		)
		n.Notify(errors.New("oops"), nil)
		n.Close()

		Expect(buf.String()).To(
			ContainSubstring("invalid project id or key"),
		)
	})

	It("logs errors on invalid host", func() {
		origLogger := gobrake.GetLogger()
		defer func() {
			gobrake.SetLogger(origLogger)
		}()

		buf := new(bytes.Buffer)
		l := log.New(buf, "", 0)
		gobrake.SetLogger(l)

		n := gobrake.NewNotifierWithOptions(
			&gobrake.NotifierOptions{
				ProjectId:  1,
				ProjectKey: "abc",
				Host:       "http://localhost:1234",
			},
		)
		n.Notify(errors.New("oops"), nil)
		n.Close()

		Expect(buf.String()).To(
			ContainSubstring("connect: connection refused"),
		)
	})
})

var _ = Describe("Deprecated filter keys option", func() {
	var deprecatedKeysOption = []interface{}{
		"thingOne",
		"thingTwo",
	}
	var notifier *gobrake.Notifier
	var sentNotice *gobrake.Notice

	var opt *gobrake.NotifierOptions

	BeforeEach(func() {
		handler := func(w http.ResponseWriter, req *http.Request) {
			b, err := ioutil.ReadAll(req.Body)
			if err != nil {
				panic(err)
			}

			sentNotice = new(gobrake.Notice)
			err = json.Unmarshal(b, sentNotice)
			Expect(err).To(BeNil())

			w.WriteHeader(http.StatusCreated)
			_, err = w.Write([]byte(`{"id":"123"}`))
			Expect(err).To(BeNil())
		}
		server := httptest.NewServer(http.HandlerFunc(handler))

		opt = &gobrake.NotifierOptions{
			ProjectId:     1,
			ProjectKey:    "key",
			Host:          server.URL,
			KeysBlacklist: deprecatedKeysOption,
		}
	})

	It("applies filters", func() {
		origLogger := gobrake.GetLogger()
		defer func() {
			gobrake.SetLogger(origLogger)
		}()

		buf := new(bytes.Buffer)
		l := log.New(buf, "", 0)
		gobrake.SetLogger(l)

		notifier = gobrake.NewNotifierWithOptions(opt)

		notice := &gobrake.Notice{
			Errors: []gobrake.Error{{
				Type:    "type1",
				Message: "msg1",
			}},
			Env: map[string]interface{}{
				"thingOne": "please filter this",
				"thingTwo": "and this too",
				"email":    "john@example.com",
			},
		}
		notifier.Notify(notice, nil)
		notifier.Flush()

		e := sentNotice.Errors[0]
		Expect(e.Type).To(Equal("type1"))
		Expect(e.Message).To(Equal("msg1"))
		Expect(sentNotice.Env).To(Equal(map[string]interface{}{
			"thingOne": "[Filtered]",
			"thingTwo": "[Filtered]",
			"email":    "john@example.com",
		}))
		Expect(buf.String()).To(
			ContainSubstring("KeysBlacklist is a deprecated option. Use KeysBlocklist instead."),
		)
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

		notifier = gobrake.NewNotifierWithOptions(&gobrake.NotifierOptions{
			ProjectId:  1,
			ProjectKey: "key",
			Host:       server.URL,
		})
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

var _ = Describe("Notice exceeds 64KB", func() {
	var notifier *gobrake.Notifier

	const maxNoticeLen = 64 * 1024

	BeforeEach(func() {
		handler := func(w http.ResponseWriter, req *http.Request) {
			w.WriteHeader(http.StatusCreated)
		}
		server := httptest.NewServer(http.HandlerFunc(handler))

		notifier = gobrake.NewNotifierWithOptions(&gobrake.NotifierOptions{
			ProjectId:  1,
			ProjectKey: "key",
			Host:       server.URL,
		})
	})

	AfterEach(func() {
		Expect(notifier.Close()).NotTo(HaveOccurred())
	})

	It("returns notice too big error", func() {
		b := make([]byte, maxNoticeLen+1)
		_, err := rand.Read(b)
		Expect(err).NotTo(HaveOccurred())

		notice := notifier.Notice(string(b), nil, 3)
		_, err = notifier.SendNotice(notice)
		Expect(err).To(MatchError("gobrake: notice exceeds 64KB max size limit"))
	})
})

var _ = Describe("server returns HTTP 400 error message", func() {
	var notifier *gobrake.Notifier

	BeforeEach(func() {
		handler := func(w http.ResponseWriter, req *http.Request) {
			w.WriteHeader(http.StatusBadRequest)
			_, err := w.Write([]byte(`{"message":"error 400!"}`))
			Expect(err).To(BeNil())
		}
		server := httptest.NewServer(http.HandlerFunc(handler))

		notifier = gobrake.NewNotifierWithOptions(&gobrake.NotifierOptions{
			ProjectId:  1,
			ProjectKey: "key",
			Host:       server.URL,
		})
	})

	AfterEach(func() {
		Expect(notifier.Close()).NotTo(HaveOccurred())
	})

	It("returns notice too big error", func() {
		notice := notifier.Notice("hello", nil, 3)
		_, err := notifier.SendNotice(notice)
		Expect(err).To(MatchError("error 400!"))
	})
})

var _ = Describe("Notifier request filter", func() {
	type routeStat struct {
		Method     string
		Route      string
		StatusCode int
		Count      int     `json:"count"`
		Sum        float64 `json:"sum"`
		Sumsq      float64 `json:"sumsq"`
		TDigest    []byte  `json:"tdigest"`
	}

	type routeStats struct {
		Routes []routeStat `json:"routes"`
	}

	var notifier *gobrake.Notifier
	var stats *routeStats

	BeforeEach(func() {
		stats = new(routeStats)

		handler := func(w http.ResponseWriter, req *http.Request) {
			b, err := ioutil.ReadAll(req.Body)
			Expect(err).NotTo(HaveOccurred())

			err = json.Unmarshal(b, &stats)
			Expect(err).NotTo(HaveOccurred())

			w.WriteHeader(http.StatusCreated)
		}
		server := httptest.NewServer(http.HandlerFunc(handler))

		notifier = gobrake.NewNotifierWithOptions(&gobrake.NotifierOptions{
			ProjectId:  1,
			ProjectKey: "key",
			Host:       server.URL,
		})

		notifier.Routes.AddFilter(func(info *gobrake.RouteMetric) *gobrake.RouteMetric {
			if info.Route == "/pong" {
				return nil
			}
			return info
		})
	})

	It("sends route stat with route is /ping", func() {
		_, metric := gobrake.NewRouteMetric(context.TODO(), "GET", "/ping")
		metric.StatusCode = http.StatusOK
		err := notifier.Routes.Notify(context.TODO(), metric)
		Expect(err).NotTo(HaveOccurred())

		notifier.Routes.Flush()
		Expect(stats.Routes).To(HaveLen(1))

		route := stats.Routes[0]
		Expect(route.Method).To(Equal("GET"))
		Expect(route.Route).To(Equal("/ping"))
		Expect(route.StatusCode).To(Equal(200))
		Expect(route.Count).To(Equal(1))
	})

	It("ignores route stat with route is /pong", func() {
		_, metric := gobrake.NewRouteMetric(context.TODO(), "GET", "/pong")
		metric.StatusCode = http.StatusOK
		err := notifier.Routes.Notify(context.TODO(), metric)
		Expect(err).NotTo(HaveOccurred())

		notifier.Routes.Flush()
		Expect(stats.Routes).To(HaveLen(0))
	})
})
