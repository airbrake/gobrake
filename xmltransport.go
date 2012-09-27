package gobrake

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"net/http"
	"reflect"
)

type XMLTransport struct{}

func NewXMLTransport() Transporter {
	return &XMLTransport{}
}

func (t *XMLTransport) fullCreateNoticeURL(n Notifier) string {
	return scheme(n) + n.CreateNoticeURL()
}

func (t *XMLTransport) marshal(n Notifier, err error, r *http.Request) ([]byte, error) {
	stack := stack(3)
	backtrace := make([]xmlBacktrace, 0, len(stack))
	for _, s := range stack {
		backtrace = append(backtrace, xmlBacktrace{s.File, s.Line, s.Func})
	}

	xmln := &xmlNotice{
		Version: "2.0",

		APIKey: n.APIKey(),
		Notifier: xmlNotifier{
			Name:    n.Name(),
			Version: n.Version(),
			URL:     n.URL(),
		},
		Error: xmlError{
			Type:      reflect.TypeOf(err).String(),
			Message:   err.Error(),
			Backtrace: backtrace,
		},
		ServerEnv: xmlServerEnv{
			EnvName:    n.EnvName(),
			AppRoot:    n.AppRoot(),
			AppVersion: n.AppVersion(),
		},
	}
	if r != nil {
		xmln.Request = xmlRequest{
			URL:       r.URL.String(),
			Component: "",
			Action:    "",
			CGIData:   []xmlVar{{"METHOD", r.Method}},
		}
		if r.RemoteAddr != "" {
			xmln.Request.CGIData = append(
				xmln.Request.CGIData, xmlVar{"REMOTE_ADDR", r.RemoteAddr})
		}
		if ua := r.Header.Get("User-Agent"); ua != "" {
			xmln.Request.CGIData = append(
				xmln.Request.CGIData, xmlVar{"HTTP_USER_AGENT", ua})
		}
	}

	b, err := xml.Marshal(xmln)
	if err != nil {
		return nil, err
	}

	// Go currently ignores omitempty on CGIData.
	b = bytes.Replace(b, []byte("<cgi-data></cgi-data>"), []byte{}, -1)

	return b, nil
}

func (t *XMLTransport) Transport(n Notifier, err error, r *http.Request) error {
	b, err := t.marshal(n, err, r)
	if err != nil {
		return err
	}

	buf := bytes.NewBufferString(xml.Header)
	buf.Write(b)

	resp, err := http.Post(t.fullCreateNoticeURL(n), "text/xml", buf)
	if err != nil {
		return err
	}
	resp.Body.Close()

	if code := resp.StatusCode; code != http.StatusOK {
		return fmt.Errorf("gobrake: got %v response, expected 200 OK", code)
	}

	return nil
}

type xmlNotifier struct {
	Name    string `xml:"name"`
	Version string `xml:"version"`
	URL     string `xml:"url"`
}

type xmlError struct {
	Type      string         `xml:"class"`
	Message   string         `xml:"message"`
	Backtrace []xmlBacktrace `xml:"backtrace>line"`
}

type xmlBacktrace struct {
	File string `xml:"file,attr"`
	Line int    `xml:"number,attr"`
	Func string `xml:"method,attr"`
}

type xmlRequest struct {
	URL       string   `xml:"url"`
	Component string   `xml:"component"`
	Action    string   `xml:"action"`
	CGIData   []xmlVar `xml:"cgi-data>var,omitempty"`
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

	APIKey    string       `xml:"api-key"`
	Notifier  xmlNotifier  `xml:"notifier"`
	Error     xmlError     `xml:"error"`
	Request   xmlRequest   `xml:"request"`
	ServerEnv xmlServerEnv `xml:"server-environment"`
}
