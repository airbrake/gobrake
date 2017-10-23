package gobrake

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

const defaultAirbrakeHost = "https://api.airbrake.io"
const waitTimeout = 5 * time.Second

const httpEnhanceYourCalm = 420
const httpStatusTooManyRequests = 429

var (
	errClosed             = errors.New("gobrake: notifier is closed")
	errUnauthorized       = errors.New("gobrake: unauthorized: invalid project id or key")
	errAccountRateLimited = errors.New("gobrake: account is rate limited")
	errIPRateLimited      = errors.New("gobrake: IP is rate limited")
)

var httpClient = &http.Client{
	Transport: &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		Dial: (&net.Dialer{
			Timeout:   15 * time.Second,
			KeepAlive: 30 * time.Second,
		}).Dial,
		TLSHandshakeTimeout: 10 * time.Second,
		TLSClientConfig: &tls.Config{
			ClientSessionCache: tls.NewLRUClientSessionCache(1024),
		},
		MaxIdleConnsPerHost:   10,
		ResponseHeaderTimeout: 10 * time.Second,
	},
	Timeout: 10 * time.Second,
}

var buffers = sync.Pool{
	New: func() interface{} {
		return new(bytes.Buffer)
	},
}

type filter func(*Notice) *Notice

type Notifier struct {
	// http.Client that is used to interact with Airbrake API.
	Client *http.Client

	projectId       int64
	projectKey      string
	createNoticeURL string

	filters []filter

	wg       sync.WaitGroup
	noticeCh chan *Notice

	rateLimitReset int64 // atomic

	_closed uint32 // atomic
	close   chan struct{}
}

func NewNotifier(projectId int64, projectKey string) *Notifier {
	n := &Notifier{
		Client: httpClient,

		projectId:       projectId,
		projectKey:      projectKey,
		createNoticeURL: buildCreateNoticeURL(defaultAirbrakeHost, projectId, projectKey),

		filters: []filter{noticeBacktraceFilter},

		noticeCh: make(chan *Notice, 1000),
		close:    make(chan struct{}),
	}
	for i := 0; i < 2*runtime.NumCPU(); i++ {
		go n.worker()
	}
	return n
}

// Sets Airbrake host name. Default is https://airbrake.io.
func (n *Notifier) SetHost(h string) {
	n.createNoticeURL = buildCreateNoticeURL(h, n.projectId, n.projectKey)
}

// AddFilter adds filter that can modify or ignore notice.
func (n *Notifier) AddFilter(fn filter) {
	n.filters = append(n.filters, fn)
}

// Notify notifies Airbrake about the error.
func (n *Notifier) Notify(e interface{}, req *http.Request) {
	notice := n.Notice(e, req, 1)
	n.SendNoticeAsync(notice)
}

// Notice returns Aibrake notice created from error and request. depth
// determines which call frame to use when constructing backtrace.
func (n *Notifier) Notice(err interface{}, req *http.Request, depth int) *Notice {
	return NewNotice(err, req, depth+3)
}

type sendResponse struct {
	Id string `json:"id"`
}

// SendNotice sends notice to Airbrake.
func (n *Notifier) SendNotice(notice *Notice) (string, error) {
	if n.closed() {
		return "", errClosed
	}

	for _, fn := range n.filters {
		notice = fn(notice)
		if notice == nil {
			// Notice is ignored.
			return "", nil
		}
	}

	if time.Now().Unix() < atomic.LoadInt64(&n.rateLimitReset) {
		return "", errIPRateLimited
	}

	buf := buffers.Get().(*bytes.Buffer)
	defer buffers.Put(buf)

	buf.Reset()
	if err := json.NewEncoder(buf).Encode(notice); err != nil {
		return "", err
	}

	resp, err := n.Client.Post(n.createNoticeURL, "application/json", buf)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	buf.Reset()
	_, err = buf.ReadFrom(resp.Body)
	if err != nil {
		return "", err
	}

	switch resp.StatusCode {
	case http.StatusCreated:
		var sendResp sendResponse
		err = json.NewDecoder(buf).Decode(&sendResp)
		if err != nil {
			return "", err
		}
		return sendResp.Id, nil
	case http.StatusUnauthorized:
		return "", errUnauthorized
	case httpStatusTooManyRequests:
		delayStr := resp.Header.Get("X-RateLimit-Delay")
		delay, err := strconv.ParseInt(delayStr, 10, 64)
		if err == nil {
			atomic.StoreInt64(&n.rateLimitReset, time.Now().Unix()+delay)
		}
		return "", errIPRateLimited
	case httpEnhanceYourCalm:
		return "", errAccountRateLimited
	}

	err = fmt.Errorf("got response status=%q, wanted 201 CREATED", resp.Status)
	logger.Printf("SendNotice failed reporting notice=%q: %s", notice, err)
	return "", err
}

// SendNoticeAsync is like SendNotice, but sends notice asynchronously.
// Pending notices can be flushed with Flush.
func (n *Notifier) SendNoticeAsync(notice *Notice) {
	if n.closed() {
		notice.Error = errClosed
		return
	}

	n.wg.Add(1)
	select {
	case n.noticeCh <- notice:
	default:
		n.wg.Done()
		logger.Printf(
			"notice=%q is ignored, because queue is full (len=%d)",
			notice, len(n.noticeCh),
		)
	}
}

func (n *Notifier) worker() {
	for {
		select {
		case notice := <-n.noticeCh:
			notice.Id, notice.Error = n.SendNotice(notice)
			n.wg.Done()
		case <-n.close:
			select {
			case notice := <-n.noticeCh:
				notice.Id, notice.Error = n.SendNotice(notice)
				n.wg.Done()
			default:
				return
			}
		}
	}
}

// NotifyOnPanic notifies Airbrake about the panic and should be used
// with defer statement.
func (n *Notifier) NotifyOnPanic() {
	if v := recover(); v != nil {
		notice := n.Notice(v, nil, 3)
		n.SendNotice(notice)
		panic(v)
	}
}

// Flush waits for pending requests to finish.
func (n *Notifier) Flush() {
	n.waitTimeout(waitTimeout)
}

func (n *Notifier) Close() error {
	return n.CloseTimeout(waitTimeout)
}

// CloseTimeout waits for pending requests to finish and then closes the notifier.
func (n *Notifier) CloseTimeout(timeout time.Duration) error {
	if !atomic.CompareAndSwapUint32(&n._closed, 0, 1) {
		return nil
	}
	err := n.waitTimeout(timeout)
	close(n.close)
	return err
}

func (n *Notifier) closed() bool {
	return atomic.LoadUint32(&n._closed) == 1
}

func (n *Notifier) waitTimeout(timeout time.Duration) error {
	done := make(chan struct{})
	go func() {
		n.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		return nil
	case <-time.After(timeout):
		return fmt.Errorf("Wait timed out after %s", timeout)
	}
}

func buildCreateNoticeURL(host string, projectId int64, key string) string {
	return fmt.Sprintf(
		"%s/api/v3/projects/%d/notices?key=%s",
		host, projectId, key,
	)
}

func noticeBacktraceFilter(notice *Notice) *Notice {
	v, ok := notice.Context["rootDirectory"]
	if !ok {
		return notice
	}

	dir, ok := v.(string)
	if !ok {
		return notice
	}

	dir = filepath.Join(dir, "src")
	for i := range notice.Errors {
		replaceRootDirectory(notice.Errors[i].Backtrace, dir)
	}
	return notice
}

func replaceRootDirectory(backtrace []StackFrame, rootDir string) {
	for i := range backtrace {
		frame := &backtrace[i]
		frame.File = strings.Replace(frame.File, rootDir, "[PROJECT_ROOT]", 1)
	}
}
