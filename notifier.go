package gobrake

import (
	"bufio"
	"bytes"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"os"
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

const maxNoticeLen = 64 * 1024

var (
	errClosed             = errors.New("gobrake: notifier is closed")
	errQueueFull          = errors.New("gobrake: queue is full (error is dropped)")
	errUnauthorized       = errors.New("gobrake: unauthorized: invalid project id or key")
	errAccountRateLimited = errors.New("gobrake: account is rate limited")
	errIPRateLimited      = errors.New("gobrake: IP is rate limited")
	errNoticeTooBig       = errors.New("gobrake: notice exceeds 64KB max size limit")
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

	inFlight int32 // atomic
	limit    chan struct{}
	wg       sync.WaitGroup

	rateLimitReset int64 // atomic

	_closed uint32 // atomic
}

func NewNotifier(projectId int64, projectKey string) *Notifier {
	n := &Notifier{
		Client: httpClient,

		projectId:       projectId,
		projectKey:      projectKey,
		createNoticeURL: buildCreateNoticeURL(defaultAirbrakeHost, projectId),

		filters: []filter{gopathFilter, gitRevisionFilter},

		limit: make(chan struct{}, 2*runtime.NumCPU()),
	}
	return n
}

// Sets Airbrake host name. Default is https://airbrake.io.
func (n *Notifier) SetHost(h string) {
	n.createNoticeURL = buildCreateNoticeURL(h, n.projectId)
}

// AddFilter adds filter that can modify or ignore notice.
func (n *Notifier) AddFilter(fn func(*Notice) *Notice) {
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

	if buf.Len() > maxNoticeLen {
		return "", errNoticeTooBig
	}

	req, err := http.NewRequest("POST", n.createNoticeURL, buf)
	if err != nil {
		return "", err
	}

	req.Header.Set("Authorization", "Bearer "+n.projectKey)
	req.Header.Set("Content-Type", "application/json")
	resp, err := n.Client.Do(req)
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

	inFlight := atomic.AddInt32(&n.inFlight, 1)
	if inFlight > 1000 {
		atomic.AddInt32(&n.inFlight, -1)
		notice.Error = errQueueFull
		return
	}

	n.wg.Add(1)
	go func() {
		n.limit <- struct{}{}

		notice.Id, notice.Error = n.SendNotice(notice)
		atomic.AddInt32(&n.inFlight, -1)
		n.wg.Done()

		<-n.limit
	}()
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

func buildCreateNoticeURL(host string, projectId int64) string {
	return fmt.Sprintf("%s/api/v3/projects/%d/notices", host, projectId)
}

func gopathFilter(notice *Notice) *Notice {
	s, ok := notice.Context["gopath"].(string)
	if !ok {
		return notice
	}

	dirs := filepath.SplitList(s)
	for i := range notice.Errors {
		backtrace := notice.Errors[i].Backtrace
		for j := range backtrace {
			frame := &backtrace[j]

			for _, dir := range dirs {
				dir = filepath.Join(dir, "src")
				if strings.HasPrefix(frame.File, dir) {
					frame.File = strings.Replace(frame.File, dir, "/GOPATH", 1)
					break
				}
			}
		}
	}

	return notice
}

func gitRevisionFilter(notice *Notice) *Notice {
	rootDir, _ := notice.Context["rootDirectory"].(string)
	rev, _ := notice.Context["revision"].(string)
	if rootDir == "" || rev != "" {
		return notice
	}

	rev, err := gitRevision(rootDir)
	if err != nil {
		return notice
	}

	notice.Context["revision"] = rev
	return notice
}

var (
	mu        sync.RWMutex
	revisions = make(map[string]interface{})
)

func gitRevision(dir string) (string, error) {
	mu.RLock()
	v := revisions[dir]
	mu.RUnlock()

	switch v := v.(type) {
	case error:
		return "", v
	case string:
		return v, nil
	}

	mu.Lock()
	defer mu.Unlock()

	rev, err := _gitRevision(dir)
	if err != nil {
		logger.Printf("gitRevision dir=%q failed: %s", dir, err)
		revisions[dir] = err
		return "", err
	}

	revisions[dir] = rev
	return rev, nil
}

func _gitRevision(dir string) (string, error) {
	head, err := gitHead(dir)
	if err != nil {
		return "", err
	}

	prefix := []byte("ref: ")
	if !bytes.HasPrefix(head, prefix) {
		return string(head), nil
	}
	ref := head[len(prefix):]

	refFile := filepath.Join(dir, ".git", string(ref))
	rev, err := ioutil.ReadFile(refFile)
	if err == nil {
		return string(trimnl(rev)), nil
	}

	refsFile := filepath.Join(dir, ".git", "packed-refs")
	fd, err := os.Open(refsFile)
	if err != nil {
		return "", err
	}

	scanner := bufio.NewScanner(fd)
	for scanner.Scan() {
		b := scanner.Bytes()
		if len(b) == 0 || b[0] == '#' || b[0] == '^' {
			continue
		}

		bs := bytes.Split(b, []byte{' '})
		if len(bs) != 2 {
			continue
		}

		if bytes.Equal(bs[1], head) {
			return string(bs[0]), nil
		}
	}
	if err := scanner.Err(); err != nil {
		return "", err
	}

	return "", fmt.Errorf("git revision for ref=%q not found", ref)
}

func gitHead(dir string) ([]byte, error) {
	headFile := filepath.Join(dir, ".git", "HEAD")
	b, err := ioutil.ReadFile(headFile)
	if err != nil {
		return nil, err
	}
	return trimnl(b), nil
}

func trimnl(b []byte) []byte {
	for _, c := range []byte{'\n', '\r'} {
		if len(b) > 0 && b[len(b)-1] == c {
			b = b[:len(b)-1]
		} else {
			break
		}
	}
	return b
}
