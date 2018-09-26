package gobrake

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"
)

const flushPeriod = 15 * time.Second

type routeKey struct {
	Method     string    `json:"method"`
	Route      string    `json:"route"`
	StatusCode int       `json:"statusCode"`
	Time       time.Time `json:"time"`
}

type routeStat struct {
	Count int     `json:"count"`
	Sum   float64 `json:"sum"`
	Sumsq float64 `json:"sumsq"`
	Min   float64 `json:"min"`
	Max   float64 `json:"max"`
}

type routeKeyStat struct {
	*routeKey
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
		apiURL: fmt.Sprintf("%s/api/v4/projects/%d/routes-stats",
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
		routes = append(routes, routeKeyStat{
			routeKey:  &k,
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

	log.Println(string(buf.Bytes()))

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

	switch resp.StatusCode {
	case http.StatusOK:
		return nil
	case http.StatusUnauthorized:
		return errUnauthorized
	}

	err = fmt.Errorf("got response status=%q, wanted 200 OK", resp.Status)
	return err
}

func (s *routeStats) IncRequest(
	method, route string, statusCode int, dur time.Duration, tm time.Time,
) error {
	key := routeKey{
		Method:     method,
		Route:      route,
		StatusCode: statusCode,
		Time:       tm.UTC().Truncate(time.Minute),
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.init()

	stat, ok := s.m[key]
	if !ok {
		stat = new(routeStat)
		s.m[key] = stat
	}

	stat.Count++
	ms := float64(dur) / float64(time.Millisecond)
	stat.Sum += ms
	stat.Sumsq += ms * ms
	if ms < stat.Min || stat.Min == 0 {
		stat.Min = ms
	}
	if ms > stat.Max {
		stat.Max = ms
	}

	return nil
}
