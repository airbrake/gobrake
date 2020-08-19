package gobrake

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"
)

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
	}
}

func (rc *remoteConfig) Poll() {
	_, err := fetchConfig(rc.configURL())
	if err != nil {
		logger.Printf(fmt.Sprintf("fetchConfig failed: %s", err))
	}

	rc.ticker = time.NewTicker(10 * time.Minute)
	go func() {
		for {
			<-rc.ticker.C
			_, err := fetchConfig(rc.configURL())
			if err != nil {
				logger.Printf(fmt.Sprintf("fetchConfig failed: %s", err))
				continue
			}
		}
	}()
}

func (rc *remoteConfig) StopPolling() {
	rc.ticker.Stop()
}

func (rc *remoteConfig) configURL() string {
	return fmt.Sprintf("%s/2020-06-18/config/%d/config.json",
		rc.opt.RemoteConfigHost, rc.opt.ProjectId)
}

func fetchConfig(url string) (*RemoteConfigJSON, error) {
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
		cfg, err := parseConfig(body)
		if err != nil {
			return nil, err
		}
		return cfg, nil
	default:
		return nil, fmt.Errorf("unhandled status (%d): %s",
			resp.StatusCode, body)
	}
}

func parseConfig(body []byte) (*RemoteConfigJSON, error) {
	var j *RemoteConfigJSON
	err := json.Unmarshal(body, &j)
	if err != nil {
		return nil, err
	}
	return j, nil
}
