package gobrake

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"net/http"
	"reflect"
)

var (
	createNoticeAPIV2URL = "//collect.airbrake.io/notifier_api/v2/notices"
)

type XMLTransport struct {
	CreateAPIURL string
	Client       *http.Client
	key          string
}

func NewXMLTransport(client *http.Client, key string, isSecure bool) *XMLTransport {
	url := scheme(isSecure) + createNoticeAPIV2URL
	return &XMLTransport{
		CreateAPIURL: url,
		Client:       client,

		key: key,
	}
}

func (t *XMLTransport) Transport(
	e error,
	stack []*stackEntry,
	r *http.Request,
	context map[string]string,
	session map[string]interface{},
) error {
	xmln := t.newXMLNotice(e, stack, r, context, session)

	buf := bytes.NewBufferString(xml.Header)
	enc := xml.NewEncoder(buf)

	if err := enc.Encode(xmln); err != nil {
		return err
	}

	// Go currently ignores omitempty on CGIData.
	// b = bytes.Replace(b, []byte("<cgi-data></cgi-data>"), []byte{}, 1)

	resp, err := t.Client.Post(t.CreateAPIURL, "text/xml", buf)
	if err != nil {
		return err
	}
	resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf(
			"gobrake: got %v response, expected 200 OK", resp.StatusCode)
	}

	return nil
}

type xmlNotifier struct {
	Name    string `xml:"name"`
	Version string `xml:"version"`
	URL     string `xml:"url"`
}

type xmlError struct {
	Type      string        `xml:"class"`
	Message   string        `xml:"message"`
	Backtrace []*stackEntry `xml:"backtrace>line"`
}

type xmlRequest struct {
	URL       string    `xml:"url"`
	Component string    `xml:"component"`
	Action    string    `xml:"action"`
	Env       []*xmlVar `xml:"cgi-data>var,omitempty"`
	Params    []*xmlVar `xml:"params>var"`
}

type xmlVar struct {
	Key   string `xml:"key,attr"`
	Value string `xml:",chardata"`
}

// Order of the fields matters.
type xmlServerEnv struct {
	AppRoot    string `xml:"project-root"`
	EnvName    string `xml:"environment-name"`
	AppVersion string `xml:"app-version"`
}

type xmlNotice struct {
	XMLName xml.Name `xml:"notice"`
	Version string   `xml:"version,attr"`

	Key       string        `xml:"api-key"`
	Notifier  *xmlNotifier  `xml:"notifier"`
	Error     *xmlError     `xml:"error"`
	Request   *xmlRequest   `xml:"request"`
	ServerEnv *xmlServerEnv `xml:"server-environment"`
}

func (t *XMLTransport) newXMLNotice(
	e error,
	stack []*stackEntry,
	r *http.Request,
	context map[string]string,
	session map[string]interface{},
) *xmlNotice {
	xmln := &xmlNotice{
		Version: "2.0",

		Key: t.key,
		Notifier: &xmlNotifier{
			Name:    "Airbrake GO XML Notifier",
			Version: notifierVersion,
			URL:     notifierURL,
		},
		Error: &xmlError{
			Type:      reflect.TypeOf(e).String(),
			Message:   e.Error(),
			Backtrace: stack,
		},
		ServerEnv: &xmlServerEnv{
			EnvName:    context["environment"],
			AppRoot:    context["rootDirectory"],
			AppVersion: context["version"],
		},
	}

	if r != nil {
		xmln.Request = &xmlRequest{
			URL:       r.URL.String(),
			Component: "",
			Action:    "",
		}

		env := make([]*xmlVar, 0, len(r.Header))
		for k, v := range r.Header {
			env = append(env, &xmlVar{k, v[0]})
		}
		xmln.Request.Env = env

		if err := r.ParseForm(); err == nil {
			params := make([]*xmlVar, 0, len(r.Form))
			for k, _ := range r.Form {
				params = append(params, &xmlVar{k, r.Form.Get(k)})
			}
			xmln.Request.Params = params
		}
	}

	return xmln
}
