package gobrake

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"runtime"
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestGobrake(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "gobrake")
}

func TestCreateNoticeURL(t *testing.T) {
	notifier := NewNotifier(1, "key")
	wanted := "https://airbrake.io/api/v3/projects/1/notices?key=key"
	if notifier.createNoticeURL != wanted {
		t.Fatalf("got %q, wanted %q", notifier.createNoticeURL, wanted)
	}
}

var _ = Describe("Notifier", func() {
	var notifier *Notifier
	var notice *Notice

	BeforeEach(func() {
		handler := func(w http.ResponseWriter, req *http.Request) {
			b, err := ioutil.ReadAll(req.Body)
			if err != nil {
				panic(err)
			}

			notice = &Notice{}
			err = json.Unmarshal(b, notice)
			Expect(err).To(BeNil())

			w.WriteHeader(http.StatusCreated)
		}
		server := httptest.NewServer(http.HandlerFunc(handler))

		notifier = NewNotifier(1, "key")
		notifier.createNoticeURL = server.URL
	})

	It("reports error and backtrace", func() {
		err := notifier.Notify("hello", nil)
		Expect(err).To(BeNil())

		e := notice.Errors[0]
		Expect(e.Type).To(Equal("string"))
		Expect(e.Message).To(Equal("hello"))
		Expect(e.Backtrace[0].File).To(ContainSubstring("notifier_test.go"))
	})

	It("reports context, env, session and params", func() {
		wanted := notifier.Notice("hello", nil, 3)
		wanted.Context["context1"] = "context1"
		wanted.Env["env1"] = "value1"
		wanted.Session["session1"] = "value1"
		wanted.Params["param1"] = "value1"
		err := notifier.SendNotice(wanted)
		Expect(err).To(BeNil())
		Expect(notice).To(Equal(wanted))
	})

	It("reports context using SetContext", func() {
		notifier.SetContext("environment", "production")
		err := notifier.Notify("hello", nil)
		Expect(err).To(BeNil())
		Expect(notice.Context["environment"]).To(Equal("production"))
	})

	It("reports request", func() {
		u, err := url.Parse("http://foo/bar")
		Expect(err).To(BeNil())

		req := &http.Request{
			URL: u,
			Header: http.Header{
				"h1":         {"h1v1", "h1v2"},
				"h2":         {"h2v1"},
				"User-Agent": {"my_user_agent"},
			},
			Form: url.Values{
				"f1": {"f1v1"},
				"f2": {"f2v1", "f2v2"},
			},
		}

		err = notifier.Notify("hello", req)
		Expect(err).To(BeNil())

		ctx := notice.Context
		Expect(ctx["url"]).To(Equal("http://foo/bar"))
		Expect(ctx["userAgent"]).To(Equal("my_user_agent"))

		params := notice.Params
		Expect(params["f1"]).To(Equal("f1v1"))
		Expect(params["f2"]).To(Equal([]interface{}{"f2v1", "f2v2"}))

		env := notice.Env
		Expect(env["h1"]).To(Equal([]interface{}{"h1v1", "h1v2"}))
		Expect(env["h2"]).To(Equal("h2v1"))
	})

	It("collects and reports context", func() {
		err := notifier.Notify("hello", nil)
		Expect(err).To(BeNil())

		hostname, _ := os.Hostname()
		wd, _ := os.Getwd()
		Expect(notice.Context["language"]).To(Equal(runtime.Version()))
		Expect(notice.Context["os"]).To(Equal(runtime.GOOS))
		Expect(notice.Context["architecture"]).To(Equal(runtime.GOARCH))
		Expect(notice.Context["hostname"]).To(Equal(hostname))
		Expect(notice.Context["rootDirectory"]).To(Equal(wd))
	})
})
