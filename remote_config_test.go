package gobrake

import (
	"bytes"
	"log"
	"net/http"
	"net/http/httptest"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("newRemoteConfig", func() {
	var rc *remoteConfig
	var opt *NotifierOptions
	var origLogger *log.Logger
	var logBuf *bytes.Buffer

	BeforeEach(func() {
		opt = &NotifierOptions{
			ProjectId:  1,
			ProjectKey: "key",
		}

		origLogger = GetLogger()
		logBuf = new(bytes.Buffer)
		SetLogger(log.New(logBuf, "", 0))
	})

	JustBeforeEach(func() {
		rc = newRemoteConfig(opt)
	})

	AfterEach(func() {
		SetLogger(origLogger)
		rc.StopPolling()
	})

	Describe("Poll", func() {
		Context("when the server returns 404", func() {
			BeforeEach(func() {
				handler := func(w http.ResponseWriter, req *http.Request) {
					w.WriteHeader(http.StatusNotFound)
					_, err := w.Write([]byte("not found"))
					Expect(err).To(BeNil())
				}
				server := httptest.NewServer(http.HandlerFunc(handler))

				opt.RemoteConfigHost = server.URL
			})

			It("logs the error", func() {
				rc.Poll()
				Expect(logBuf.String()).To(
					ContainSubstring("fetchConfig failed: not found"),
				)
			})
		})

		Context("when the server returns 403", func() {
			BeforeEach(func() {
				handler := func(w http.ResponseWriter, req *http.Request) {
					w.WriteHeader(http.StatusForbidden)
					_, err := w.Write([]byte("forbidden"))
					Expect(err).To(BeNil())
				}
				server := httptest.NewServer(http.HandlerFunc(handler))

				opt.RemoteConfigHost = server.URL
			})

			It("logs the error", func() {
				rc.Poll()
				Expect(logBuf.String()).To(
					ContainSubstring("fetchConfig failed: forbidden"),
				)
			})
		})

		Context("when the server returns 200", func() {
			BeforeEach(func() {
				handler := func(w http.ResponseWriter, req *http.Request) {
					w.WriteHeader(http.StatusOK)
					_, err := w.Write([]byte("{}"))
					Expect(err).To(BeNil())
				}
				server := httptest.NewServer(http.HandlerFunc(handler))

				opt.RemoteConfigHost = server.URL
			})

			It("doesn't log any errors", func() {
				rc.Poll()
				Expect(logBuf.String()).To(BeEmpty())
			})
		})

		Context("when the server returns unhandled code", func() {
			BeforeEach(func() {
				handler := func(w http.ResponseWriter, req *http.Request) {
					w.WriteHeader(http.StatusGone)
					_, err := w.Write([]byte("{}"))
					Expect(err).To(BeNil())
				}
				server := httptest.NewServer(http.HandlerFunc(handler))

				opt.RemoteConfigHost = server.URL
			})

			It("logs the unhandled error", func() {
				rc.Poll()
				Expect(logBuf.String()).To(
					ContainSubstring("unhandled status (410): {}"),
				)
			})
		})
	})
})
