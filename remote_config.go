package gobrake

import (
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"
)

type remoteConfig struct {
	opt    *NotifierOptions
	ticker *time.Ticker
}

func newRemoteConfig(opt *NotifierOptions) *remoteConfig {
	return &remoteConfig{
		opt: opt,
	}
}

func (rc *remoteConfig) Poll() {
	err := rc.fetchConfig()
	if err != nil {
		logger.Printf(fmt.Sprintf("fetchConfig failed: %s", err))
	}

	rc.ticker = time.NewTicker(10 * time.Minute)
	go func() {
		for {
			<-rc.ticker.C
			err := rc.fetchConfig()
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

func (rc *remoteConfig) fetchConfig() error {
	url := fmt.Sprintf("%s/2020-06-18/config/%d/config.json",
		rc.opt.RemoteConfigHost, rc.opt.ProjectId)
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	switch resp.StatusCode {
	case http.StatusForbidden, http.StatusNotFound:
		return errors.New(string(body))
	case http.StatusOK:
		return nil
	default:
		return fmt.Errorf("unhandled status (%d): %s",
			resp.StatusCode, body)
	}
}
