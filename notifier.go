package gobrake

import (
	"bytes"
	"context"
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

const notifierName = "gobrake"
const notifierVersion = "4.2.0"
const userAgent = notifierName + "/" + notifierVersion

const waitTimeout = 5 * time.Second

const httpEnhanceYourCalm = 420
const httpStatusTooManyRequests = 429

const maxNoticeLen = 64 * 1024

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

	// Airbrake host name. Default is https://airbrake.io.
	Host string

	// Airbrake host name for sending APM data.
	APMHost string

	// Environment such as production or development.
	Environment string

	// Git revision. Default is SOURCE_VERSION on Heroku.
	Revision string

	// List of keys containing sensitive information that must be filtered out.
	// Default is password, secret.
	KeysBlocklist []interface{}

	// Deprecated version of "KeysBlocklist". Still supported but will eventually
	// be removed in a future release.
	KeysBlacklist []interface{}

	// Disables code hunks.
	DisableCodeHunks bool

	// Controls the error reporting feature.
	DisableErrorNotifications bool

	// Controls the error reporting feature.
	DisableAPM bool

	// http.Client that is used to interact with Airbrake API.
	HTTPClient *http.Client
}

func (opt *NotifierOptions) init() {
	if opt.Host == "" {
		opt.Host = "https://api.airbrake.io"
	}

	if opt.APMHost == "" {
		opt.APMHost = opt.Host
	}

	if opt.Revision == "" {
		// https://devcenter.heroku.com/changelog-items/630
		opt.Revision = os.Getenv("SOURCE_VERSION")
	}

	if len(opt.KeysBlacklist) > 0 {
		opt.KeysBlocklist = opt.KeysBlacklist
		logger.Printf("KeysBlacklist is a deprecated option. Use KeysBlocklist instead.")
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

type routes struct {
	filters []routeFilter

	stats      *routeStats
	breakdowns *routeBreakdowns
}

func newRoutes(opt *NotifierOptions) *routes {
	return &routes{
		stats:      newRouteStats(opt),
		breakdowns: newRouteBreakdowns(opt),
	}
}

// AddFilter adds filter that can change route stat or ignore it by returning nil.
func (rs *routes) AddFilter(fn func(*RouteMetric) *RouteMetric) {
	rs.filters = append(rs.filters, fn)
}

func (rs *routes) Flush() {
	rs.stats.Flush()
	rs.breakdowns.Flush()
}

func (rs *routes) Notify(c context.Context, metric *RouteMetric) error {
	metric.finish()

	for _, fn := range rs.filters {
		metric = fn(metric)
		if metric == nil {
			return nil
		}
	}

	err := rs.stats.Notify(c, metric)
	if err != nil {
		return err
	}

	err = rs.breakdowns.Notify(c, metric)
	if err != nil {
		return err
	}

	return nil
}

type Notifier struct {
	opt             *NotifierOptions
	createNoticeURL string

	filters []filter

	inFlight int32 // atomic
	limit    chan struct{}
	wg       sync.WaitGroup

	Routes  *routes
	Queries *queryStats
	Queues  *queueStats

	rateLimitReset uint32 // atomic
	_closed        uint32 // atomic
}

func NewNotifierWithOptions(opt *NotifierOptions) *Notifier {
	opt.init()

	n := &Notifier{
		opt: opt,
		createNoticeURL: fmt.Sprintf("%s/api/v3/projects/%d/notices",
			opt.Host, opt.ProjectId),

		limit: make(chan struct{}, 2*runtime.NumCPU()),

		Routes:  newRoutes(opt),
		Queries: newQueryStats(opt),
		Queues:  newQueueStats(opt),
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

	req, err := http.NewRequest("POST", n.createNoticeURL, buf)
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
	case httpStatusTooManyRequests:
		delayStr := resp.Header.Get("X-RateLimit-Delay")
		delay, err := strconv.ParseInt(delayStr, 10, 64)
		if err == nil {
			atomic.StoreUint32(&n.rateLimitReset, uint32(time.Now().Unix()+delay))
		}
		return "", errIPRateLimited
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
	}

	err = fmt.Errorf("got unexpected response status=%q", resp.Status)
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
			return
		}

		panic(v)
	}
}

// Flush waits for pending requests to finish.
func (n *Notifier) Flush() {
	_ = n.waitTimeout(waitTimeout)
}

func (n *Notifier) Close() error {
	return n.CloseTimeout(waitTimeout)
}

// CloseTimeout waits for pending requests to finish and then closes the notifier.
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
		return fmt.Errorf("Wait timed out after %s", timeout)
	}
}
