package gobrake

import (
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
)

const (
	notifierName    = "Airbrake Go notifier"
	notifierVersion = "1.0"
	notifierURL     = "http://github.com/airbrake/gobrake"
)

var (
	deployURL       = "://go-airbrake.appspot.com/deploys.txt"
	createNoticeURL = "://go-airbrake.appspot.com/notifier_api/v2/notices"

	Logger = log.New(os.Stderr, "", log.LstdFlags)
)

type Transporter interface {
	Transport(Notifier, error, *http.Request) error
}

type Notifier interface {
	Name() string
	Version() string
	URL() string

	APIKey() string
	EnvName() string
	AppVersion() string
	AppRoot() string
	IsSecure() bool

	DeployURL() string
	CreateNoticeURL() string

	Notify(error, *http.Request) error
	Deploy(string, string, string) error
}

type Config struct {
	APIKey     string
	EnvName    string
	AppVersion string
	AppRoot    string
	IsSecure   bool

	Transport Transporter

	DeployURL       string
	CreateNoticeURL string
}

func NewNotifier(c Config) Notifier {
	if c.APIKey == "" {
		panic("gobrake: API key is empty")
	}
	if c.AppRoot == "" {
		if wd, err := os.Getwd(); err == nil {
			c.AppRoot = wd
		}
	}
	if c.Transport == nil {
		c.Transport = NewXMLTransport()
	}
	if c.DeployURL == "" {
		c.DeployURL = deployURL
	}
	if c.CreateNoticeURL == "" {
		c.CreateNoticeURL = createNoticeURL
	}

	return &StdNotifier{
		name:    notifierName,
		version: notifierVersion,
		url:     notifierURL,
		config:  &c,
	}
}

type StdNotifier struct {
	name    string
	version string
	url     string
	config  *Config
}

func (n *StdNotifier) Name() string    { return n.name }
func (n *StdNotifier) Version() string { return n.version }
func (n *StdNotifier) URL() string     { return n.url }

func (n *StdNotifier) APIKey() string     { return n.config.APIKey }
func (n *StdNotifier) EnvName() string    { return n.config.EnvName }
func (n *StdNotifier) AppVersion() string { return n.config.AppVersion }
func (n *StdNotifier) AppRoot() string    { return n.config.AppRoot }
func (n *StdNotifier) IsSecure() bool     { return n.config.IsSecure }

func (n *StdNotifier) DeployURL() string       { return n.config.DeployURL }
func (n *StdNotifier) CreateNoticeURL() string { return n.config.CreateNoticeURL }

func (n *StdNotifier) Notify(e error, r *http.Request) error {
	if e == nil {
		Logger.Printf("gobrake: error is nil")
		return nil
	}

	if err := n.config.Transport.Transport(n, e, r); err != nil {
		Logger.Printf("gobrake error: %v", err)
		return err
	}
	return nil
}

func (n *StdNotifier) fullDeployURL() string {
	return scheme(n) + n.config.DeployURL
}

func (n *StdNotifier) Deploy(repository, revision, username string) error {
	req, err := http.NewRequest("POST", n.fullDeployURL(), nil)
	if err != nil {
		return err
	}
	req.Form = url.Values{
		"api_key":                {n.APIKey()},
		"deploy[rails_env]":      {n.EnvName()},
		"deploy[scm_repository]": {repository},
		"deploy[scm_revision]":   {revision},
		"deploy[local_username]": {username},
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if code := resp.StatusCode; code != http.StatusOK {
		return fmt.Errorf("gobrake: got %v response, expected 200 OK", code)
	}

	return nil
}
