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
	notifierURL     = "http://github.com/airbrake/goairbrake"
)

var (
	deployURL = "//go-airbrake.appspot.com/deploys.txt"

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
	SetIsSecure(bool)

	Notify(error, *http.Request) error
	Deploy(string, string, string) error
}

func NewNotifierTransport(apiKey, envName, appVersion, appRoot string, transport Transporter) Notifier {
	if apiKey == "" {
		panic("goairbrake: apiKey is empty")
	}

	if appRoot == "" {
		if wd, err := os.Getwd(); err == nil {
			appRoot = wd
		}
	}

	return &StdNotifier{
		name:    notifierName,
		version: notifierVersion,
		url:     notifierURL,

		apiKey:     apiKey,
		envName:    envName,
		appVersion: appVersion,
		appRoot:    appRoot,

		transport: transport,

		deployURL: deployURL,
	}
}

func NewNotifier(apiKey, envName, appVersion, appRoot string) Notifier {
	return NewNotifierTransport(apiKey, envName, appVersion, appRoot, NewXMLTransport())
}

type StdNotifier struct {
	name    string
	version string
	url     string

	apiKey     string
	envName    string
	appVersion string
	appRoot    string

	transport Transporter

	secure    bool
	deployURL string
}

func (n *StdNotifier) Name() string    { return n.name }
func (n *StdNotifier) Version() string { return n.version }
func (n *StdNotifier) URL() string     { return n.url }

func (n *StdNotifier) SetIsSecure(secure bool) { n.secure = secure }
func (n *StdNotifier) IsSecure() bool          { return n.secure }

func (n *StdNotifier) APIKey() string     { return n.apiKey }
func (n *StdNotifier) EnvName() string    { return n.envName }
func (n *StdNotifier) AppVersion() string { return n.appVersion }
func (n *StdNotifier) AppRoot() string    { return n.appRoot }

func (n *StdNotifier) Notify(e error, r *http.Request) error {
	if e == nil {
		Logger.Printf("goairbrake: error is nil")
		return nil
	}

	if err := n.transport.Transport(n, e, r); err != nil {
		Logger.Printf("goairbrake error: %v", err)
		return err
	}
	return nil
}

func (n *StdNotifier) fullDeployURL() string {
	return proto(n) + n.deployURL
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
		return fmt.Errorf("goairbrake: got %v response, expected 200 OK", code)
	}

	return nil
}
