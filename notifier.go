package gobrake // import "gopkg.in/airbrake/gobrake.v2"

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"runtime"
	"sync"
	"time"
)

type filter func(*Notice) *Notice

var httpClient = &http.Client{
	Transport: &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		Dial: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
		}).Dial,
		TLSHandshakeTimeout: 10 * time.Second,
		TLSClientConfig: &tls.Config{
			ClientSessionCache: tls.NewLRUClientSessionCache(1024),
		},
		MaxIdleConnsPerHost:   100,
		ResponseHeaderTimeout: 10 * time.Second,
	},
	Timeout: 10 * time.Second,
}

type Notifier struct {
	Client          *http.Client
	createNoticeURL string
	context         map[string]string
	filters         []filter
	wg              sync.WaitGroup
}

func NewNotifier(projectId int64, projectKey string) *Notifier {
	n := &Notifier{
		Client:          httpClient,
		createNoticeURL: getCreateNoticeURL(projectId, projectKey),
		context: map[string]string{
			"language":     runtime.Version(),
			"os":           runtime.GOOS,
			"architecture": runtime.GOARCH,
		},
	}
	if hostname, err := os.Hostname(); err == nil {
		n.context["hostname"] = hostname
	}
	if wd, err := os.Getwd(); err == nil {
		n.context["rootDirectory"] = wd
	}
	return n
}

// AddFilter adds filter that can modify or ignore notice.
func (n *Notifier) AddFilter(fn filter) {
	n.filters = append(n.filters, fn)
}

// Notify notifies Airbrake about the error.
func (n *Notifier) Notify(e interface{}, req *http.Request) {
	notice := n.Notice(e, req, 1)
	n.wg.Add(1)
	go func() {
		if _, err := n.SendNotice(notice); err != nil {
			log.Printf("gobrake failed (%s) reporting error: %v", err, e)
		}
		n.wg.Done()
	}()
}

// Notice returns Aibrake notice created from error and request. depth
// determines which call frame to use.
func (n *Notifier) Notice(err interface{}, req *http.Request, depth int) *Notice {
	notice := NewNotice(err, req, depth+3)
	for k, v := range n.context {
		notice.Context[k] = v
	}
	return notice
}

type sendResponse struct {
	Id string `json:"id"`
}

// SendNotice sends notice to Airbrake.
func (n *Notifier) SendNotice(notice *Notice) (string, error) {
	for _, fn := range n.filters {
		notice = fn(notice)
		if notice == nil {
			// Notice is ignored.
			return "", nil
		}
	}

	buf := &bytes.Buffer{}
	enc := json.NewEncoder(buf)
	if err := enc.Encode(notice); err != nil {
		return "", err
	}

	resp, err := n.Client.Post(n.createNoticeURL, "application/json", buf)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	if resp.StatusCode != http.StatusCreated {
		return "", fmt.Errorf("gobrake: got %d response, wanted 201 CREATED", resp.StatusCode)
	}

	var sendResp sendResponse
	err = json.Unmarshal(b, &sendResp)
	if err != nil {
		return "", err
	}

	return sendResp.Id, nil
}

// Flush flushes all pending I/O.
func (n *Notifier) Flush() {
	n.wg.Wait()
}
