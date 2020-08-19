package gobrake

import (
	"bytes"
	"log"
	"net/http"
	"net/http/httptest"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("newRemoteConfig", func() {
	var rc *remoteConfig
	var opt *NotifierOptions
	var origLogger *log.Logger
	var logBuf *bytes.Buffer

	Describe("Poll", func() {
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
			Context("and when it returns correct config JSON", func() {
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

			Context("and when it returns incorrect JSON config", func() {
				BeforeEach(func() {
					handler := func(w http.ResponseWriter, req *http.Request) {
						w.WriteHeader(http.StatusOK)
						_, err := w.Write([]byte("{"))
						Expect(err).To(BeNil())
					}
					server := httptest.NewServer(http.HandlerFunc(handler))

					opt.RemoteConfigHost = server.URL
				})

				It("logs the error", func() {
					rc.Poll()
					Expect(logBuf.String()).To(
						ContainSubstring(
							"parseConfig failed: unexpected end of JSON input",
						),
					)
				})
			})

			Context("and when it returns JSON with missing config fields", func() {
				BeforeEach(func() {
					handler := func(w http.ResponseWriter, req *http.Request) {
						w.WriteHeader(http.StatusOK)
						_, err := w.Write([]byte(`{"hello":"hi"}`))
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

			Context("and when it returns JSON with current config fields", func() {
				var body = `{"project_id":1,"updated_at":2,` +
					`"poll_sec":3,"config_route":"abc/config.json",` +
					`"settings":[{"name":"errors","enabled":false,` +
					`"endpoint":null}]}`

				BeforeEach(func() {
					handler := func(w http.ResponseWriter, req *http.Request) {
						w.WriteHeader(http.StatusOK)
						_, err := w.Write([]byte(body))
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

	Describe("Interval", func() {
		BeforeEach(func() {
			rc = newRemoteConfig(&NotifierOptions{
				ProjectId:  1,
				ProjectKey: "key",
			})
		})

		Context("when JSON PollSec is zero", func() {
			JustBeforeEach(func() {
				rc.JSON.PollSec = 0
			})

			It("returns the default interval", func() {
				Expect(rc.Interval()).To(Equal(600 * time.Second))
			})
		})

		Context("when JSON PollSec less than zero", func() {
			JustBeforeEach(func() {
				rc.JSON.PollSec = -123
			})

			It("returns the default interval", func() {
				Expect(rc.Interval()).To(Equal(600 * time.Second))
			})
		})

		Context("when JSON PollSec is above zero", func() {
			BeforeEach(func() {
				rc.JSON.PollSec = 123
			})

			It("returns the interval from JSON", func() {
				Expect(rc.Interval()).To(Equal(123 * time.Second))
			})
		})
	})
})
