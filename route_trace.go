package gobrake

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"
)

type ctxKey string

const traceCtxKey ctxKey = "ab_route_trace"

type routeBreakdownKey struct {
	Method   string    `json:"method"`
	Route    string    `json:"route"`
	RespType string    `json:"responseType"`
	Time     time.Time `json:"time"`
}

type routeBreakdown struct {
	routeBreakdownKey

	routeStat
	Groups map[string]*routeStat `json:"groups"`
}

func (b *routeBreakdown) Add(total time.Duration, groups map[string]time.Duration) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.Groups == nil {
		b.Groups = make(map[string]*routeStat)
	}

	_ = b.routeStat.Add(total)

	if groups == nil {
		groups = make(map[string]time.Duration)
	}

	var sum time.Duration
	for name, dur := range groups {
		sum += dur

		s, ok := b.Groups[name]
		if !ok {
			s = newRouteStat()
			b.Groups[name] = s
		}
		_ = s.Add(dur)
	}

	if sum > total {
		logger.Printf("sum=%s > total=%s", sum, total)
	}
}

func (b *routeBreakdown) Pack() error {
	err := b.routeStat.Pack()
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
	if trace.StatusCode < 200 || (trace.StatusCode >= 300 && trace.StatusCode < 400) {
		// ignore
		return nil
	}

	key := routeBreakdownKey{
		Method:   trace.Method,
		Route:    trace.Route,
		RespType: trace.respType(),
		Time:     trace.Start.UTC().Truncate(time.Minute),
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
	addWG.Add(1)
	s.mu.Unlock()

	total := trace.End.Sub(trace.Start)
	groups := trace.flushGroups()

	b.Add(total, groups)
	addWG.Done()

	return nil
}

type RouteTrace struct {
	Method      string
	Route       string
	StatusCode  int
	ContentType string

	// TODO: unexport and add a mutex
	Start time.Time
	End   time.Time

	spansMu  sync.Mutex
	spans    map[string]*span
	currSpan *span

	groupsMu sync.Mutex
	groups   map[string]time.Duration
}

func NewRouteTrace(c context.Context, trace *RouteTrace) (context.Context, *RouteTrace) {
	if trace.Start.IsZero() {
		trace.Start = clock.Now()
	}
	c = context.WithValue(c, traceCtxKey, trace)
	return c, trace
}

func RouteTraceFromContext(c context.Context) *RouteTrace {
	if c == nil {
		return nil
	}
	t, _ := c.Value(traceCtxKey).(*RouteTrace)
	return t
}

func (t *RouteTrace) StartSpan(name string) {
	if t == nil {
		return
	}

	t.spansMu.Lock()
	defer t.spansMu.Unlock()

	if t.spans == nil {
		t.spans = make(map[string]*span)
	}

	if t.currSpan != nil && t.currSpan.name == name {
		t.currSpan.level++
		return
	}

	t.currSpan.pause()

	span, ok := t.spans[name]
	if ok {
		span.resume()
	} else {
		span = newSpan(t, name)
		t.spans[name] = span
	}

	span.parent = t.currSpan
	t.currSpan = span
}

func (t *RouteTrace) EndSpan(name string) {
	if t == nil {
		return
	}

	t.spansMu.Lock()
	defer t.spansMu.Unlock()

	if t.currSpan != nil && t.currSpan.name == name {
		if t.endSpan(t.currSpan) {
			t.currSpan = t.currSpan.parent
			t.currSpan.resume()
		}
		return
	}

	span, ok := t.spans[name]
	if !ok {
		logger.Printf("gobrake: span=%q does not exist", name)
		return
	}
	t.endSpan(span)
}

func (t *RouteTrace) endSpan(span *span) bool {
	if span.level > 0 {
		span.level--
		return false
	}
	span.End()
	delete(t.spans, span.name)
	return true
}

func (t *RouteTrace) incGroup(name string, dur time.Duration) {
	t.groupsMu.Lock()
	if !t.End.IsZero() {
		return
	}
	if t.groups == nil {
		t.groups = make(map[string]time.Duration)
	}
	t.groups[name] += dur
	t.groupsMu.Unlock()
}

func (t *RouteTrace) flushGroups() map[string]time.Duration {
	t.groupsMu.Lock()
	groups := t.groups
	t.groups = nil
	t.groupsMu.Unlock()
	return groups
}

func (t *RouteTrace) respType() string {
	if t.StatusCode >= 400 {
		return "error"
	}
	if t.ContentType == "" {
		return ""
	}
	ind := strings.LastIndexByte(t.ContentType, '/')
	if ind != -1 {
		return t.ContentType[ind+1:]
	}
	return t.ContentType
}

type Span interface {
	End()
}

type span struct {
	trace  *RouteTrace
	parent *span

	name  string
	start time.Time
	dur   time.Duration
	level int
}

var _ Span = (*span)(nil)

func newSpan(trace *RouteTrace, name string) *span {
	return &span{
		trace: trace,
		name:  name,
		start: clock.Now(),
	}
}

func (s *span) End() {
	s.dur += clock.Since(s.start)
	s.trace.incGroup(s.name, s.dur)
	s.trace = nil
}

func (s *span) pause() {
	if s == nil || s.paused() {
		return
	}
	s.dur += clock.Since(s.start)
	s.start = time.Time{}
}

func (s *span) paused() bool {
	return s.start.IsZero()
}

func (s *span) resume() {
	if s == nil || !s.paused() {
		return
	}
	s.start = clock.Now()
}

type noopSpan struct{}

var _ Span = noopSpan{}

func (noopSpan) End() {}

func durInMs(dur time.Duration) float64 {
	return float64(dur) / float64(time.Millisecond)
}
