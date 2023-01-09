package gobrake

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sync"
	"time"
)

type routeKey struct {
	Method     string    `json:"method"`
	Route      string    `json:"route"`
	StatusCode int       `json:"statusCode"`
	Time       time.Time `json:"time"`
}

type routeKeyStat struct {
	routeKey
	*tdigestStat
}

// routeStats aggregates information about requests and periodically sends
// collected data to Airbrake.
type routeStats struct {
	opt        *NotifierOptions
	flushTimer *time.Timer
	addWG      *sync.WaitGroup

	mu sync.Mutex
	m  map[routeKey]*tdigestStat
}

type routeFilter func(*RouteMetric) *RouteMetric

func newRouteStats(opt *NotifierOptions) *routeStats {
	return &routeStats{
		opt: opt,
	}
}

func (s *routeStats) init() {
	if s.flushTimer == nil {
		s.flushTimer = time.AfterFunc(flushPeriod, s.Flush)
		s.addWG = new(sync.WaitGroup)
		s.m = make(map[routeKey]*tdigestStat)
	}
}

// Flush sends to Airbrake route stats.
func (s *routeStats) Flush() {
	s.mu.Lock()

	s.flushTimer = nil
	addWG := s.addWG
	s.addWG = nil
	m := s.m
	s.m = nil

	s.mu.Unlock()

	if m == nil {
		return
	}

	addWG.Wait()
	err := s.send(m)
	if err != nil {
		logger.Printf("routeStats.send failed: %s", err)
	}
}

type routesOut struct {
	Env    string         `json:"environment"`
	Routes []routeKeyStat `json:"routes"`
}

func (s *routeStats) send(m map[routeKey]*tdigestStat) error {
	var routes []routeKeyStat
	for k, v := range m {
		err := v.Pack()
		if err != nil {
			return err
		}

		routes = append(routes, routeKeyStat{
			routeKey:    k,
			tdigestStat: v,
		})
	}

	buf := buffers.Get().(*bytes.Buffer)
	defer buffers.Put(buf)
	buf.Reset()

	out := routesOut{
		Env:    s.opt.Environment,
		Routes: routes,
	}
	err := json.NewEncoder(buf).Encode(out)
	if err != nil {
		return err
	}

	req, err := http.NewRequest(
		http.MethodPut,
		fmt.Sprintf("%s/api/v5/projects/%d/routes-stats",
			s.opt.APMHost, s.opt.ProjectId),
		buf,
	)
	if err != nil {
		return err
	}

	req.Header.Set("Authorization", "Bearer "+s.opt.ProjectKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", userAgent)
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
	case http.StatusBadRequest, http.StatusTooManyRequests:
		var sendResp sendResponse
		err = json.NewDecoder(buf).Decode(&sendResp)
		if err != nil {
			return err
		}
		return errors.New(sendResp.Message)
	case 404, 408, 409, 410, 500, 502, 504:
		setRouteStatBacklog(out)
	}

	err = fmt.Errorf("got unexpected response status=%q", resp.Status)
	return err
}

// Notify adds new route stats.
func (s *routeStats) Notify(c context.Context, req *RouteMetric) error {
	if s.opt.DisableAPM {
		return fmt.Errorf(
			"APM is disabled, route is not sent: %s %s (status %d)",
			req.Method, req.Route, req.StatusCode,
		)
	}

	key := routeKey{
		Method:     req.Method,
		Route:      req.Route,
		StatusCode: req.StatusCode,
		Time:       req.startTime.UTC().Truncate(time.Minute),
	}

	s.mu.Lock()
	s.init()
	stat, ok := s.m[key]
	if !ok {
		stat = newTDigestStat()
		s.m[key] = stat
	}
	addWG := s.addWG
	addWG.Add(1)
	s.mu.Unlock()

	dur := req.endTime.Sub(req.startTime)
	err := stat.Add(dur)
	addWG.Done()

	return err
}
