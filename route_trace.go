package gobrake

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"
)

type ctxKey string

const traceCtxKey ctxKey = "ab_route_trace"

type routeBreakdownKey struct {
	Method      string    `json:"method"`
	Route       string    `json:"route"`
	ContentType string    `json:"contentType"`
	Time        time.Time `json:"time"`
}

type routeBreakdown struct {
	routeBreakdownKey

	mu     sync.Mutex
	Total  *routeStat            `json:"total"`
	Groups map[string]*routeStat `json:"groups"`
}

func (b *routeBreakdown) Add(total float64, groups map[string]float64) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.Total == nil {
		b.Total = newRouteStat()
	}
	if b.Groups == nil {
		b.Groups = make(map[string]*routeStat)
	}

	_ = b.Total.Add(total)

	var sum float64
	for _, ms := range groups {
		sum += ms
	}

	other := total - sum
	if other < 0 {
		other = 0
	}

	if groups == nil {
		groups = make(map[string]float64)
	}
	groups["other"] = other

	for name, ms := range groups {
		s, ok := b.Groups[name]
		if !ok {
			s = newRouteStat()
			b.Groups[name] = s
		}
		_ = s.Add(ms)
	}
}

func (b *routeBreakdown) Pack() error {
	err := b.Total.Pack()
	if err != nil {
		return err
	}

	for _, v := range b.Groups {
		err = v.Pack()
		if err != nil {
			return err
		}
	}

	return nil
}

type routeBreakdowns struct {
	opt    *NotifierOptions
	apiURL string

	flushTimer *time.Timer
	addWG      *sync.WaitGroup

	mu sync.Mutex
	m  map[routeBreakdownKey]*routeBreakdown
}

func newRouteBreakdowns(opt *NotifierOptions) *routeBreakdowns {
	return &routeBreakdowns{
		opt: opt,
		apiURL: fmt.Sprintf("%s/api/v5/projects/%d/routes-breakdowns",
			opt.Host, opt.ProjectId),
	}
}

func (s *routeBreakdowns) init() {
	if s.flushTimer == nil {
		s.flushTimer = time.AfterFunc(flushPeriod, s.Flush)
		s.addWG = new(sync.WaitGroup)
		s.m = make(map[routeBreakdownKey]*routeBreakdown)
	}
}

// Flush sends to Airbrake route stats.
func (s *routeBreakdowns) Flush() {
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
		logger.Printf("routeBreakdowns.send failed: %s", err)
	}
}

type breakdownsOut struct {
	Env    string            `json:"environment"`
	Routes []*routeBreakdown `json:"routes"`
}

func (s *routeBreakdowns) send(m map[routeBreakdownKey]*routeBreakdown) error {
	var routes []*routeBreakdown
	for _, v := range m {
		err := v.Pack()
		if err != nil {
			return err
		}
		routes = append(routes, v)
	}

	buf := buffers.Get().(*bytes.Buffer)
	defer buffers.Put(buf)
	buf.Reset()

	out := breakdownsOut{
		Env:    s.opt.Environment,
		Routes: routes,
	}
	err := json.NewEncoder(buf).Encode(out)
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

func (s *routeBreakdowns) Notify(c context.Context, trace *RouteTrace) error {
	key := routeBreakdownKey{
		Method:      trace.Method,
		Route:       trace.Route,
		ContentType: trace.ContentType,
		Time:        trace.Start.UTC().Truncate(time.Minute),
	}

	s.mu.Lock()
	s.init()
	b, ok := s.m[key]
	if !ok {
		b = &routeBreakdown{
			routeBreakdownKey: key,
		}
		s.m[key] = b
	}
	addWG := s.addWG
	s.addWG.Add(1)
	s.mu.Unlock()

	total := durInMs(trace.End.Sub(trace.Start))
	trace.mu.Lock()
	groups := trace.groups
	trace.groups = nil
	trace.mu.Unlock()

	s.mu.Lock()
	b.Add(total, groups)
	addWG.Done()
	s.mu.Unlock()

	return nil
}

type RouteTrace struct {
	Method      string
	Route       string
	StatusCode  int
	ContentType string

	Start time.Time
	End   time.Time

	mu     sync.Mutex
	groups map[string]float64
}

func NewRouteTrace(c context.Context, trace *RouteTrace) (context.Context, *RouteTrace) {
	if trace.Start.IsZero() {
		trace.Start = time.Now()
	}
	c = context.WithValue(c, traceCtxKey, trace)
	return c, trace
}

func RouteTraceFromContext(c context.Context) *RouteTrace {
	t, _ := c.Value(traceCtxKey).(*RouteTrace)
	return t
}

func (t *RouteTrace) Group(name string) Group {
	s := &group{
		trace: t,
		name:  name,
		start: time.Now(),
	}
	return s
}

func (t *RouteTrace) IncGroup(name string, ms float64) {
	t.mu.Lock()
	if t.groups == nil {
		t.groups = make(map[string]float64)
	}
	t.groups[name] += ms
	t.mu.Unlock()
}

type Group interface {
	Finish()
}

type group struct {
	trace *RouteTrace
	name  string
	start time.Time
}

func (g *group) Finish() {
	since := time.Since(g.start)
	g.trace.IncGroup(g.name, durInMs(since))
}

func durInMs(dur time.Duration) float64 {
	return float64(dur) / float64(time.Millisecond)
}
