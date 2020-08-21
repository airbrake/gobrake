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

		Context("when the remote config alters poll_sec", func() {
			var body = `{"poll_sec":1}`

			BeforeEach(func() {
				handler := func(w http.ResponseWriter, req *http.Request) {
					w.WriteHeader(http.StatusOK)
					_, err := w.Write([]byte(body))
					Expect(err).To(BeNil())
				}
				server := httptest.NewServer(http.HandlerFunc(handler))

				opt.RemoteConfigHost = server.URL
			})

			It("changes interval", func() {
				Expect(rc.Interval()).NotTo(Equal(1 * time.Second))
				rc.Poll()
				rc.StopPolling()
				Expect(rc.Interval()).To(Equal(1 * time.Second))
			})
		})

		Context("when the remote config alters config_route", func() {
			var body = `{"config_route":"route/cfg.json"}`

			BeforeEach(func() {
				handler := func(w http.ResponseWriter, req *http.Request) {
					w.WriteHeader(http.StatusOK)
					_, err := w.Write([]byte(body))
					Expect(err).To(BeNil())
				}
				server := httptest.NewServer(http.HandlerFunc(handler))

				opt.RemoteConfigHost = server.URL
			})

			It("changes config route", func() {
				Expect(rc.ConfigRoute("http://example.com")).NotTo(Equal(
					"http://example.com/route/cfg.json",
				))
				rc.Poll()
				rc.StopPolling()
				Expect(rc.ConfigRoute("http://example.com")).To(Equal(
					"http://example.com/route/cfg.json",
				))
			})
		})

		Context("when the remote config enables errors", func() {
			BeforeEach(func() {
				rc.JSON.RemoteSettings = append(
					rc.JSON.RemoteSettings,
					&RemoteSettings{Name: "errors", Enabled: true},
				)
			})

			Context("and when local error notifications are disabled", func() {
				BeforeEach(func() {
					opt.DisableErrorNotifications = true
				})

				It("keeps error notifications disabled", func() {
					rc.Poll()
					rc.StopPolling()
					Expect(opt.DisableErrorNotifications).To(BeTrue())
				})
			})

			Context("and when local error notifications are enabled", func() {
				BeforeEach(func() {
					opt.DisableErrorNotifications = false
				})

				It("enables error notifications", func() {
					rc.Poll()
					rc.StopPolling()
					Expect(opt.DisableErrorNotifications).To(BeFalse())
				})
			})
		})

		Context("when the remote config enables APM", func() {
			BeforeEach(func() {
				rc.JSON.RemoteSettings = append(
					rc.JSON.RemoteSettings,
					&RemoteSettings{Name: "apm", Enabled: true},
				)
			})

			Context("and when local APM is disabled", func() {
				BeforeEach(func() {
					opt.DisableAPM = true
				})

				It("keeps APM disabled", func() {
					rc.Poll()
					rc.StopPolling()
					Expect(opt.DisableAPM).To(BeTrue())
				})
			})

			Context("and when local error notifications are enabled", func() {
				BeforeEach(func() {
					opt.DisableAPM = false
				})

				It("enables APM", func() {
					rc.Poll()
					rc.StopPolling()
					Expect(opt.DisableAPM).To(BeFalse())
				})
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

	Describe("ConfigRoute", func() {
		BeforeEach(func() {
			rc = newRemoteConfig(&NotifierOptions{
				ProjectId:  1,
				ProjectKey: "key",
			})
		})

		Context("when JSON ConfigRoute is empty", func() {
			BeforeEach(func() {
				rc.JSON.ConfigRoute = ""
			})

			It("returns the default config route", func() {
				Expect(rc.ConfigRoute("http://example.com")).To(Equal(
					"http://example.com/2020-06-18/config/1/config.json",
				))
			})
		})

		Context("when JSON ConfigRoute is non-empty", func() {
			BeforeEach(func() {
				rc.JSON.ConfigRoute = "1999/123/config.json"
			})

			It("returns the config route from JSON", func() {
				Expect(rc.ConfigRoute("http://example.com")).To(Equal(
					"http://example.com/1999/123/config.json",
				))
			})
		})

		Context("when given hostname ends with a dash", func() {
			It("trims the dash and returns the correct route", func() {
				host := "http://example.com/"
				Expect(rc.ConfigRoute(host)).To(Equal(
					"http://example.com/2020-06-18/config/1/config.json",
				))
			})
		})
	})

	Describe("ErrorNotifications", func() {
		Context("when JSON has the 'errors' setting", func() {
			BeforeEach(func() {
				rc.JSON.RemoteSettings = append(
					rc.JSON.RemoteSettings,
					&RemoteSettings{Name: "errors"},
				)
			})

			Context("and when it is enabled", func() {
				BeforeEach(func() {
					rc.JSON.RemoteSettings[0].Enabled = true
				})

				It("returns true", func() {
					Expect(rc.ErrorNotifications()).To(BeTrue())
				})
			})

			Context("and when it is disabled", func() {
				BeforeEach(func() {
					rc.JSON.RemoteSettings[0].Enabled = false
				})

				It("returns false", func() {
					Expect(rc.ErrorNotifications()).To(BeFalse())
				})
			})
		})

		Context("when JSON has NO 'errors' setting", func() {
			BeforeEach(func() {
				rc.JSON.RemoteSettings = make([]*RemoteSettings, 0)
			})

			It("returns the value from local options", func() {
				Expect(rc.ErrorNotifications()).To(BeTrue())
			})
		})
	})

	Describe("APM", func() {
		Context("when JSON has the 'apm' setting", func() {
			BeforeEach(func() {
				rc.JSON.RemoteSettings = append(
					rc.JSON.RemoteSettings,
					&RemoteSettings{Name: "apm"},
				)
			})

			Context("and when it is enabled", func() {
				BeforeEach(func() {
					rc.JSON.RemoteSettings[0].Enabled = true
				})

				It("returns true", func() {
					Expect(rc.APM()).To(BeTrue())
				})
			})

			Context("and when it is disabled", func() {
				BeforeEach(func() {
					rc.JSON.RemoteSettings[0].Enabled = false
				})

				It("returns false", func() {
					Expect(rc.APM()).To(BeFalse())
				})
			})
		})

		Context("when JSON has NO 'errors' setting", func() {
			BeforeEach(func() {
				rc.JSON.RemoteSettings = make([]*RemoteSettings, 0)
			})

			It("returns the value from local options", func() {
				Expect(rc.APM()).To(BeTrue())
			})
		})
	})
})
