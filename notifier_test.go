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

	"github.com/airbrake/gobrake/v5"
	"github.com/airbrake/gobrake/v5/internal/testpkg1"
)

func TestGobrake(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "gobrake")
}

func newConfigServer() *httptest.Server {
	handler := func(w http.ResponseWriter, req *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte(`{}`))
		Expect(err).To(BeNil())
	}
	return httptest.NewServer(http.HandlerFunc(handler))
}

func cleanupConfig() {
	configPath := "config.json"

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return
	}

	if err := os.Remove(configPath); err != nil {
		log.Fatal(err)
	}
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
		configServer := newConfigServer()

		opt = &gobrake.NotifierOptions{
			ProjectId:        1,
			ProjectKey:       "key",
			Host:             server.URL,
			RemoteConfigHost: configServer.URL,
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

	Context("DisableErrorNotifications", func() {
		Context("when it is enabled", func() {
			var origLogger *log.Logger

			BeforeEach(func() {
				origLogger = gobrake.GetLogger()
				buf := new(bytes.Buffer)
				l := log.New(buf, "", 0)
				gobrake.SetLogger(l)

				opt.DisableErrorNotifications = true
			})

			AfterEach(func() {
				gobrake.SetLogger(origLogger)

				opt.DisableErrorNotifications = false
			})

			It("doesn't send error notifications", func() {
				sentNotice = nil
				notify("hello", nil)
				Expect(sentNotice).To(BeNil())
			})

			It("logs the error", func() {
				origLogger := gobrake.GetLogger()
				defer func() {
					gobrake.SetLogger(origLogger)
				}()

				buf := new(bytes.Buffer)
				l := log.New(buf, "", 0)
				gobrake.SetLogger(l)

				n := gobrake.NewNotifierWithOptions(
					&gobrake.NotifierOptions{
						ProjectId:                 1,
						ProjectKey:                "abc",
						DisableErrorNotifications: true,
					},
				)
				n.Notify("oops", nil)
				n.Close()

				out := `error notifications are disabled, will not ` +
					`deliver notice="oops"`
				Expect(buf.String()).To(ContainSubstring(out))

			})
		})

		Context("when it is disabled", func() {
			BeforeEach(func() {
				opt.DisableErrorNotifications = false
			})

			It("sends error notifications", func() {
				sentNotice = nil
				notify("hello", nil)
				Expect(sentNotice).NotTo(BeNil())
			})
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
		Expect(sentNotice.Context["component"]).To(Equal("github.com/airbrake/gobrake/v5_test"))
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

		// Return Unauthorized for the error API.
		handler := func(w http.ResponseWriter, req *http.Request) {
			w.WriteHeader(http.StatusUnauthorized)
			_, err := w.Write([]byte(""))
			Expect(err).To(BeNil())
		}
		errorServer := httptest.NewServer(http.HandlerFunc(handler))

		configServer := newConfigServer()

		n := gobrake.NewNotifierWithOptions(
			&gobrake.NotifierOptions{
				ProjectId:        1,
				ProjectKey:       "broken-key",
				Host:             errorServer.URL,
				RemoteConfigHost: configServer.URL,
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
		configServer := newConfigServer()

		notifier = gobrake.NewNotifierWithOptions(&gobrake.NotifierOptions{
			ProjectId:        1,
			ProjectKey:       "key",
			Host:             server.URL,
			RemoteConfigHost: configServer.URL,
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
		configServer := newConfigServer()

		notifier = gobrake.NewNotifierWithOptions(&gobrake.NotifierOptions{
			ProjectId:        1,
			ProjectKey:       "key",
			Host:             server.URL,
			RemoteConfigHost: configServer.URL,
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
		configServer := newConfigServer()

		notifier = gobrake.NewNotifierWithOptions(&gobrake.NotifierOptions{
			ProjectId:        1,
			ProjectKey:       "key",
			Host:             server.URL,
			RemoteConfigHost: configServer.URL,
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
	var opt *gobrake.NotifierOptions

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
		configServer := newConfigServer()

		opt = &gobrake.NotifierOptions{
			ProjectId:        1,
			ProjectKey:       "key",
			Host:             server.URL,
			RemoteConfigHost: configServer.URL,
		}

	})

	JustBeforeEach(func() {
		notifier = gobrake.NewNotifierWithOptions(opt)
		notifier.Routes.AddFilter(func(info *gobrake.RouteMetric) *gobrake.RouteMetric {
			if info.Route == "/pong" {
				return nil
			}
			return info
		})
	})

	It("ignores route stat with route is /pong", func() {
		_, metric := gobrake.NewRouteMetric(context.TODO(), "GET", "/pong")
		metric.StatusCode = http.StatusOK
		err := notifier.Routes.Notify(context.TODO(), metric)
		Expect(err).NotTo(HaveOccurred())

		notifier.Routes.Flush()
		Expect(stats.Routes).To(HaveLen(0))
	})

	Context("when DisableAPM is enabled", func() {
		BeforeEach(func() {
			opt.DisableAPM = true
		})

		It("returns an error", func() {
			c, metric := gobrake.NewRouteMetric(context.TODO(), "GET", "/ping")
			err := notifier.Routes.Notify(c, metric)

			expectedErr := errors.New(
				"APM is disabled, route is not sent: GET " +
					"/ping (status 0)",
			)
			Expect(err).To(Equal(expectedErr))
		})
	})

	Context("when DisableAPM is disabled", func() {
		BeforeEach(func() {
			opt.DisableAPM = false
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
	})
})

var _ = Describe("(*NotifierOptions).Copy()", func() {
	var opt = &gobrake.NotifierOptions{
		ProjectId:                 1,
		ProjectKey:                "2",
		Host:                      "error host",
		APMHost:                   "apm host",
		RemoteConfigHost:          "cfg host",
		Environment:               "env",
		Revision:                  "rev",
		DisableCodeHunks:          true,
		DisableErrorNotifications: true,
		DisableAPM:                true,
	}

	It("copies ProjectId", func() {
		copy := opt.Copy()
		copy.ProjectId = 99
		Expect(opt.ProjectId).To(Equal(int64(1)))
	})

	It("copies ProjectKey", func() {
		copy := opt.Copy()
		copy.ProjectKey = "99"
		Expect(opt.ProjectKey).To(Equal("2"))
	})

	It("copies Host", func() {
		copy := opt.Copy()
		copy.Host = "aaa"
		Expect(opt.Host).To(Equal("error host"))
	})

	It("copies APMHost", func() {
		copy := opt.Copy()
		copy.APMHost = "aaa"
		Expect(opt.APMHost).To(Equal("apm host"))
	})

	It("copies RemoteConfigHost", func() {
		copy := opt.Copy()
		copy.RemoteConfigHost = "aaa"
		Expect(opt.RemoteConfigHost).To(Equal("cfg host"))
	})

	It("copies Environment", func() {
		copy := opt.Copy()
		copy.Environment = "aaa"
		Expect(opt.Environment).To(Equal("env"))
	})

	It("copies Revision", func() {
		copy := opt.Copy()
		copy.Revision = "aaa"
		Expect(opt.Revision).To(Equal("rev"))
	})

	It("copies DisableCodeHunks", func() {
		copy := opt.Copy()
		copy.DisableCodeHunks = false
		Expect(opt.DisableCodeHunks).To(BeTrue())
	})

	It("copies DisableErrorNotifications", func() {
		copy := opt.Copy()
		copy.DisableErrorNotifications = false
		Expect(opt.DisableErrorNotifications).To(BeTrue())
	})

	It("copies DisableAPM", func() {
		copy := opt.Copy()
		copy.DisableAPM = false
		Expect(opt.DisableAPM).To(BeTrue())
	})
})
