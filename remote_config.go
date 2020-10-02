package gobrake

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"runtime"
	"strings"
	"time"
)

// How frequently we should poll the config API.
const defaultInterval = 10 * time.Minute

// API version of the S3 API to poll.
const apiVer = "2020-06-18"

// What path to poll.
const configRoutePattern = "%s/%s/config/%d/config.json"

// Setting names in JSON returned by the API.
const (
	errorsSetting = "errors"
	apmSetting    = "apm"
)

type remoteConfig struct {
	opt *NotifierOptions
	// opt copy to capture the initial state of the local config.
	origOpt *NotifierOptions

	ticker   *time.Ticker
	pollStop chan bool

	JSON *RemoteConfigJSON
}

type RemoteConfigJSON struct {
	ProjectId   int64  `json:"project_id"`
	UpdatedAt   int64  `json:"updated_at"`
	PollSec     int64  `json:"poll_sec"`
	ConfigRoute string `json:"config_route"`

	RemoteSettings []*RemoteSettings `json:"settings"`
}

type RemoteSettings struct {
	Name     string `json:"name"`
	Enabled  bool   `json:"enabled"`
	Endpoint string `json:"endpoint"`
}

func newRemoteConfig(opt *NotifierOptions) *remoteConfig {
	return &remoteConfig{
		opt:     opt,
		origOpt: opt.Copy(),

		JSON: &RemoteConfigJSON{},
	}
}

func (rc *remoteConfig) Poll() {
	rc.pollStop = make(chan bool)

	go func() {
		rc.updateLocalConfig()

		if err := rc.tick(); err != nil {
			logger.Print(err)
		}
		rc.updateLocalConfig()

		rc.ticker = time.NewTicker(rc.Interval())

		for {
			select {
			case <-rc.ticker.C:
				if err := rc.tick(); err != nil {
					logger.Print(err)
					continue
				}

				rc.ticker.Stop()
				rc.updateLocalConfig()

				rc.ticker = time.NewTicker(rc.Interval())
			case <-rc.pollStop:
				break
			}
		}
	}()
}

func (rc *remoteConfig) tick() error {
	route := rc.ConfigRoute(rc.opt.RemoteConfigHost)
	body, err := rc.fetchConfig(route)
	if err != nil {
		return fmt.Errorf(
			"fetchConfig failed for %s. Reason: %s", route, err,
		)
	}
	if err = json.Unmarshal(body, rc.JSON); err != nil {
		return fmt.Errorf("parseConfig failed: %s", err)
	}

	return nil
}

func (rc *remoteConfig) updateLocalConfig() {
	if rc.ErrorHost() != "" {
		rc.opt.Host = rc.ErrorHost()
	}

	if rc.APMHost() != "" {
		rc.opt.APMHost = rc.APMHost()
	}

	rc.updateErrorNotifications()
	rc.updateAPM()
}

func (rc *remoteConfig) updateErrorNotifications() {
	if rc.origOpt.DisableErrorNotifications {
		return
	}

	rc.opt.DisableErrorNotifications = !rc.ErrorNotifications()
}

func (rc *remoteConfig) updateAPM() {
	if rc.origOpt.DisableAPM {
		return
	}

	rc.opt.DisableAPM = !rc.APM()
}

func (rc *remoteConfig) StopPolling() {
	if rc.ticker != nil {
		rc.ticker.Stop()
	}
	if rc.pollStop != nil {
		rc.pollStop <- true
	}
}

func (rc *remoteConfig) Interval() time.Duration {
	if rc.JSON.PollSec > 0 {
		return time.Duration(rc.JSON.PollSec) * time.Second
	}

	return defaultInterval
}

func (rc *remoteConfig) ConfigRoute(remoteConfigHost string) string {
	if rc.JSON.ConfigRoute != "" {
		return fmt.Sprintf("%s/%s",
			strings.TrimSuffix(remoteConfigHost, "/"),
			rc.JSON.ConfigRoute)
	}

	return fmt.Sprintf(configRoutePattern,
		strings.TrimSuffix(remoteConfigHost, "/"),
		apiVer, rc.opt.ProjectId)
}

func (rc *remoteConfig) ErrorNotifications() bool {
	for _, s := range rc.JSON.RemoteSettings {
		if s.Name == errorsSetting {
			return s.Enabled
		}
	}

	return true
}

func (rc *remoteConfig) APM() bool {
	for _, s := range rc.JSON.RemoteSettings {
		if s.Name == apmSetting {
			return s.Enabled
		}
	}

	return true
}

func (rc *remoteConfig) ErrorHost() string {
	for _, s := range rc.JSON.RemoteSettings {
		if s.Name == errorsSetting {
			return s.Endpoint
		}
	}

	return ""
}

func (rc *remoteConfig) APMHost() string {
	for _, s := range rc.JSON.RemoteSettings {
		if s.Name == apmSetting {
			return s.Endpoint
		}
	}

	return ""
}

func (rc *remoteConfig) fetchConfig(url string) ([]byte, error) {
	req, err := buildRequest(url)
	if err != nil {
		return nil, err
	}

	resp, err := rc.opt.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	switch resp.StatusCode {
	case http.StatusForbidden, http.StatusNotFound:
		return nil, errors.New(string(body))
	case http.StatusOK:
		return body, nil
	default:
		return nil, fmt.Errorf("unhandled status (%d): %s",
			resp.StatusCode, body)
	}
}

func buildRequest(url string) (*http.Request, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	q := req.URL.Query()
	q.Add("notifier_name", notifierName)
	q.Add("notifier_version", notifierVersion)
	q.Add("os", runtime.GOOS)
	q.Add("language", runtime.Version())

	req.URL.RawQuery = q.Encode()

	return req, nil
}
