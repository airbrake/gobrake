package gobrake

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"regexp"
	"runtime"
	"strconv"
	"sync"
	"sync/atomic"
	"time"
)

const (
	notifierName        = "gobrake"
	notifierVersion     = "5.6.1"
	userAgent           = notifierName + "/" + notifierVersion
	waitTimeout         = 5 * time.Second
	flushPeriod         = 15 * time.Second
	httpEnhanceYourCalm = 420
	maxNoticeLen        = 64 * 1024
)

var (
	errClosed             = errors.New("gobrake: notifier is closed")
	errQueueFull          = errors.New("gobrake: queue is full (error is dropped)")
	errUnauthorized       = errors.New("gobrake: unauthorized: invalid project id or key")
	errAccountRateLimited = errors.New("gobrake: account is rate limited")
	errIPRateLimited      = errors.New("gobrake: IP is rate limited")
	errNoticeTooBig       = errors.New("gobrake: notice exceeds 64KB max size limit")
)

var (
	httpClientOnce sync.Once
	httpClient     *http.Client
)

func defaultHTTPClient() *http.Client {
	httpClientOnce.Do(func() {
		httpClient = &http.Client{
			Timeout: 10 * time.Second,
		}
	})
	return httpClient
}

var buffers = sync.Pool{
	New: func() interface{} {
		return new(bytes.Buffer)
	},
}

type filter func(*Notice) *Notice

type NotifierOptions struct {
	// Airbrake project id.
	ProjectId int64

	// Airbrake project key.
	ProjectKey string

	// Airbrake host name. Default is https://api.airbrake.io.
	Host string

	// Airbrake host name for sending APM data.
	APMHost string

	// The host name where the remote config is located.
	RemoteConfigHost string

	// Controls the remote config feature.
	DisableRemoteConfig bool

	// Environment such as production or development.
	Environment string

	// Git revision. Default is SOURCE_VERSION on Heroku.
	Revision string

	// List of keys containing sensitive information that must be filtered out.
	// Default is password, secret.
	KeysBlocklist []interface{}

	// Disables code hunks.
	DisableCodeHunks bool

	// Controls the error reporting feature.
	DisableErrorNotifications bool

	// Controls the error reporting feature.
	DisableAPM bool

	// http.Client that is used to interact with Airbrake API.
	HTTPClient *http.Client

	// Controls the backlog reporting feature.
	// Default is false
	DisableBacklog bool
}

func (opt *NotifierOptions) init() {
	if opt.Host == "" {
		opt.Host = "https://api.airbrake.io"
	}

	if opt.APMHost == "" {
		opt.APMHost = opt.Host
	}

	if opt.RemoteConfigHost == "" {
		opt.RemoteConfigHost = "https://notifier-configs.airbrake.io"
	}

	if opt.Revision == "" {
		// https://devcenter.heroku.com/changelog-items/630
		opt.Revision = os.Getenv("SOURCE_VERSION")
	}

	if opt.KeysBlocklist == nil {
		opt.KeysBlocklist = []interface{}{
			regexp.MustCompile("password"),
			regexp.MustCompile("secret"),
		}
	}

	if opt.HTTPClient == nil {
		opt.HTTPClient = defaultHTTPClient()
	}
}

// Makes a shallow copy (without copying slices or nested structs; because we
// don't need it so far).
func (opt *NotifierOptions) Copy() *NotifierOptions {
	return &NotifierOptions{
		ProjectId:                 opt.ProjectId,
		ProjectKey:                opt.ProjectKey,
		Host:                      opt.Host,
		APMHost:                   opt.APMHost,
		RemoteConfigHost:          opt.RemoteConfigHost,
		Environment:               opt.Environment,
		Revision:                  opt.Revision,
		KeysBlocklist:             opt.KeysBlocklist,
		DisableCodeHunks:          opt.DisableCodeHunks,
		DisableErrorNotifications: opt.DisableErrorNotifications,
		DisableAPM:                opt.DisableAPM,
		HTTPClient:                opt.HTTPClient,
		DisableBacklog:            opt.DisableBacklog,
	}
}

type Notifier struct {
	opt     *NotifierOptions
	filters []filter

	inFlight int32 // atomic
	limit    chan struct{}
	wg       sync.WaitGroup

	Routes  *routes
	Queries *queryStats
	Queues  *queueStats

	rateLimitReset uint32 // atomic
	_closed        uint32 // atomic

	remoteConfig *remoteConfig
}

func NewNotifierWithOptions(opt *NotifierOptions) *Notifier {
	opt.init()

	n := &Notifier{
		opt:   opt,
		limit: make(chan struct{}, 2*runtime.NumCPU()),

		Routes:  newRoutes(opt),
		Queries: newQueryStats(opt),
		Queues:  newQueueStats(opt),

		remoteConfig: newRemoteConfig(opt),
	}

	n.AddFilter(httpUnsolicitedResponseFilter)
	n.AddFilter(newNotifierFilter(n))
	n.AddFilter(gitFilter)
	if !opt.DisableCodeHunks {
		n.AddFilter(codeHunksFilter)
	}
	n.AddFilter(gopathFilter)

	if len(opt.KeysBlocklist) > 0 {
		n.AddFilter(NewBlocklistKeysFilter(opt.KeysBlocklist...))
	}

	if !opt.DisableRemoteConfig {
		n.remoteConfig.Poll()
	}

	newBacklog(opt)
	return n
}

func NewNotifier(projectId int64, projectKey string) *Notifier {
	return NewNotifierWithOptions(&NotifierOptions{
		ProjectId:  projectId,
		ProjectKey: projectKey,
	})
}

// AddFilter adds filter that can change notice or ignore it by returning nil.
func (n *Notifier) AddFilter(fn func(*Notice) *Notice) {
	n.filters = append(n.filters, fn)
}

// Notify notifies Airbrake about the error.
func (n *Notifier) Notify(e interface{}, req *http.Request) {
	if n.opt.DisableErrorNotifications {
		logger.Printf(
			"error notifications are disabled, will not deliver notice=%q",
			e,
		)
		return
	}

	notice := n.Notice(e, req, 1)
	n.SendNoticeAsync(notice)
}

// Notice returns Aibrake notice created from error and request. depth
// determines which call frame to use when constructing backtrace.
func (n *Notifier) Notice(err interface{}, req *http.Request, depth int) *Notice {
	return NewNotice(err, req, depth+1)
}

type sendResponse struct {
	Id      string `json:"id"`
	Message string `json:"message"`
}

// SendNotice sends notice to Airbrake.
func (n *Notifier) SendNotice(notice *Notice) (string, error) {
	if n.closed() {
		return "", errClosed
	}
	return n.sendNotice(notice)
}

func (n *Notifier) sendNotice(notice *Notice) (string, error) {
	for _, fn := range n.filters {
		notice = fn(notice)
		if notice == nil {
			// Notice is ignored.
			return "", nil
		}
	}

	if time.Now().Unix() < int64(atomic.LoadUint32(&n.rateLimitReset)) {
		return "", errIPRateLimited
	}

	buf := buffers.Get().(*bytes.Buffer)
	defer buffers.Put(buf)

	buf.Reset()
	err := json.NewEncoder(buf).Encode(notice)
	if err != nil {
		return "", err
	}

	if buf.Len() > maxNoticeLen {
		return "", errNoticeTooBig
	}

	req, err := http.NewRequest(
		http.MethodPost,
		fmt.Sprintf("%s/api/v3/projects/%d/notices",
			n.opt.Host, n.opt.ProjectId),
		buf,
	)
	if err != nil {
		return "", err
	}

	req.Header.Set("Authorization", "Bearer "+n.opt.ProjectKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", userAgent)
	resp, err := n.opt.HTTPClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	buf.Reset()
	_, err = buf.ReadFrom(resp.Body)
	if err != nil {
		return "", err
	}

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		var sendResp sendResponse
		err = json.NewDecoder(buf).Decode(&sendResp)
		if err != nil {
			return "", err
		}
		return sendResp.Id, nil
	}

	switch resp.StatusCode {
	case http.StatusUnauthorized:
		return "", errUnauthorized
	case http.StatusTooManyRequests:
		delayStr := resp.Header.Get("X-RateLimit-Delay")
		delay, err := strconv.ParseInt(delayStr, 10, 64)
		if err == nil {
			atomic.StoreUint32(&n.rateLimitReset, uint32(time.Now().Unix()+delay))
		}
		var sendResp sendResponse
		err = json.NewDecoder(buf).Decode(&sendResp)
		if err != nil {
			return "", err
		}
		return "", errors.New(sendResp.Message)
	case httpEnhanceYourCalm:
		return "", errAccountRateLimited
	case http.StatusRequestEntityTooLarge:
		return "", errNoticeTooBig
	case http.StatusBadRequest:
		var sendResp sendResponse
		err = json.NewDecoder(buf).Decode(&sendResp)
		if err != nil {
			return "", err
		}
		return "", errors.New(sendResp.Message)
	case 404, 408, 409, 410, 500, 502, 504:
		setNoticeBacklog(notice)
	}

	err = fmt.Errorf("got unexpected response status=%q", resp.Status)
	logger.Printf("SendNotice failed reporting notice=%q: %s", notice, err)
	return "", err
}

// SendNoticeAsync is like SendNotice, but sends notice asynchronously.
// Pending notices can be flushed with Flush().
func (n *Notifier) SendNoticeAsync(notice *Notice) {
	if n.closed() {
		notice.Error = errClosed
		return
	}

	inFlight := atomic.AddInt32(&n.inFlight, 1)
	if inFlight > 1000 {
		atomic.AddInt32(&n.inFlight, -1)
		notice.Error = errQueueFull
		return
	}

	n.wg.Add(1)
	go func() {
		n.limit <- struct{}{}

		notice.Id, notice.Error = n.sendNotice(notice)
		if notice.Error != nil {
			logger.Printf(
				"sendNotice failed reporting notice=%q: %s",
				notice, notice.Error,
			)
		}

		atomic.AddInt32(&n.inFlight, -1)
		n.wg.Done()

		<-n.limit
	}()
}

// NotifyOnPanic notifies Airbrake about the panic and should be used
// with defer statement.
func (n *Notifier) NotifyOnPanic() {
	if v := recover(); v != nil {
		notice := n.Notice(v, nil, 2)
		notice.Context["severity"] = "critical"
		_, err := n.SendNotice(notice)
		if err != nil {
			logger.Printf(
				"SendNotice failed reporting notice=%q: %s",
				notice, err,
			)
		}

		panic(v)
	}
}

// Flush waits for pending requests to finish.
// It is recommended to be used with SendNoticeAsync().
func (n *Notifier) Flush() {
	_ = n.waitTimeout(waitTimeout)
}

// Close waits for pending requests to finish and then closes the notifier.
func (n *Notifier) Close() error {
	n.remoteConfig.StopPolling()
	return n.CloseTimeout(waitTimeout)
}

// CloseTimeout waits for pending requests to finish with a custom input timeout and then closes the notifier.
func (n *Notifier) CloseTimeout(timeout time.Duration) error {
	if !atomic.CompareAndSwapUint32(&n._closed, 0, 1) {
		return nil
	}
	return n.waitTimeout(timeout)
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
		return fmt.Errorf("wait timed out after %s", timeout)
	}
}
