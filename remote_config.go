package gobrake

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"
)

// How frequently we should poll the config API.
const defaultInterval = 10 * time.Minute

type remoteConfig struct {
	opt    *NotifierOptions
	ticker *time.Ticker

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
		opt: opt,

		JSON: &RemoteConfigJSON{},
	}
}

func (rc *remoteConfig) Poll() {
	err := rc.tick()
	if err != nil {
		logger.Print(err)
	}

	rc.ticker = time.NewTicker(rc.Interval())

	go func() {
		for {
			<-rc.ticker.C
			err := rc.tick()
			if err != nil {
				logger.Print(err)
				continue
			}
			rc.ticker.Stop()
			rc.ticker = time.NewTicker(rc.Interval())
		}
	}()
}

func (rc *remoteConfig) tick() error {
	body, err := fetchConfig(rc.configURL())
	if err != nil {
		return fmt.Errorf("fetchConfig failed: %s", err)
	}

	err = json.Unmarshal(body, rc.JSON)
	if err != nil {
		return fmt.Errorf("parseConfig failed: %s", err)
	}

	return nil
}

func (rc *remoteConfig) StopPolling() {
	if rc.ticker != nil {
		rc.ticker.Stop()
	}
}

func (rc *remoteConfig) Interval() time.Duration {
	if rc.JSON.PollSec > 0 {
		return time.Duration(rc.JSON.PollSec) * time.Second
	}

	return defaultInterval
}

func (rc *remoteConfig) configURL() string {
	return fmt.Sprintf("%s/2020-06-18/config/%d/config.json",
		rc.opt.RemoteConfigHost, rc.opt.ProjectId)
}

func fetchConfig(url string) ([]byte, error) {
	resp, err := http.Get(url)
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
