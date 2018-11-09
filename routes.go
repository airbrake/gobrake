package gobrake

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	tdigest "github.com/caio/go-tdigest"
)

const flushPeriod = 15 * time.Second

type RequestInfo struct {
	Method     string
	Route      string
	StatusCode int
	Start      time.Time
	End        time.Time
}

type routeKey struct {
	Method     string    `json:"method"`
	Route      string    `json:"route"`
	StatusCode int       `json:"status_code"`
	Time       time.Time `json:"time"`
}

type routeStat struct {
	Count   int     `json:"count"`
	Sum     float64 `json:"sum"`
	Sumsq   float64 `json:"sumsq"`
	TDigest []byte  `json:"tdigest"`
	td      *tdigest.TDigest
}

func newRouteStat() *routeStat {
	td, err := tdigest.New(tdigest.Compression(20))
	if err != nil {
		panic(err)
	}
	return &routeStat{
		td: td,
	}
}

type routeKeyStat struct {
	routeKey
	*routeStat
}

// routeStats aggregates information about requests and periodically sends
// collected data to Airbrake.
type routeStats struct {
	opt    *NotifierOptions
	apiURL string

	mu sync.Mutex
	m  map[routeKey]*routeStat

	flushTimer *time.Timer
}

func newRouteStats(opt *NotifierOptions) *routeStats {
	return &routeStats{
		opt: opt,
		apiURL: fmt.Sprintf("%s/api/v5/projects/%d/routes-stats",
			opt.Host, opt.ProjectId),
	}
}

func (s *routeStats) init() {
	if s.m == nil && s.flushTimer == nil {
		s.m = make(map[routeKey]*routeStat)
		s.flushTimer = time.AfterFunc(flushPeriod, s.flush)
	}
}

func (s *routeStats) flush() {
	s.mu.Lock()

	m := s.m
	s.m = nil
	s.flushTimer = nil

	s.mu.Unlock()

	err := s.send(m)
	if err != nil {
		logger.Printf("routeStats.send failed: %s", err)
	}
}

type routesStatsJSONRequest struct {
	Routes []routeKeyStat `json:"routes"`
}

func (s *routeStats) send(m map[routeKey]*routeStat) error {
	var routes []routeKeyStat
	for k, v := range m {
		err := v.td.Compress()
		if err != nil {
			return err
		}

		b, err := v.td.AsBytes()
		if err != nil {
			return err
		}
		v.TDigest = b

		routes = append(routes, routeKeyStat{
			routeKey:  k,
			routeStat: v,
		})
	}

	jsonReq := routesStatsJSONRequest{
		Routes: routes,
	}

	buf := buffers.Get().(*bytes.Buffer)
	defer buffers.Put(buf)

	buf.Reset()
	err := json.NewEncoder(buf).Encode(jsonReq)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("PUT", s.apiURL, buf)
	if err != nil {
		return err
	}

	req.Header.Set("Authorization", "Bearer "+s.opt.ProjectKey)
	req.Header.Set("Content-Type", "application/json")
	resp, err := s.opt.HTTPClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	buf.Reset()
	_, err = buf.ReadFrom(resp.Body)
	if err != nil {
		return err
	}

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return nil
	}

	switch resp.StatusCode {
	case http.StatusUnauthorized:
		return errUnauthorized
	}

	err = fmt.Errorf("got unexpected response status=%q", resp.Status)
	return err
}

func (s *routeStats) NotifyRequest(req *RequestInfo) error {
	key := routeKey{
		Method:     req.Method,
		Route:      req.Route,
		StatusCode: req.StatusCode,
		Time:       req.Start.UTC().Truncate(time.Minute),
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.init()

	stat, ok := s.m[key]
	if !ok {
		stat = newRouteStat()
		s.m[key] = stat
	}

	stat.Count++
	ms := float64(req.End.Sub(req.Start)) / float64(time.Millisecond)
	stat.Sum += ms
	stat.Sumsq += ms * ms
	err := stat.td.Add(ms)
	if err != nil {
		return err
	}

	return nil
}
