package gobrake

import (
	"context"
	"errors"
	"net/http/httptrace"
	"sync"
	"sync/atomic"
	"time"
)

type ctxKey string

const metricCtxKey ctxKey = "ab_metric"
const spanCtxKey ctxKey = "ab_span"

type Metric interface {
	Start(c context.Context, name string) (context.Context, Span)
}

func withMetric(c context.Context, t Metric) context.Context {
	c = context.WithValue(c, metricCtxKey, t)

	var span Span
	clientTrace := &httptrace.ClientTrace{
		GetConn: func(hostPort string) {
			_, span = t.Start(c, "http.client")
		},
		GotFirstResponseByte: func() {
			span.Finish()
		},
	}
	c = httptrace.WithClientTrace(c, clientTrace)

	return c
}

func ContextMetric(c context.Context) Metric {
	if c == nil {
		return noopMetric{}
	}
	if t, ok := c.Value(metricCtxKey).(Metric); ok {
		return t
	}
	return noopMetric{}
}

func ContextSpan(c context.Context) Span {
	if c == nil {
		return noopSpan{}
	}
	if sp, ok := c.Value(spanCtxKey).(Span); ok {
		return sp
	}
	return noopSpan{}
}

//------------------------------------------------------------------------------

type noopMetric struct{}

var _ Metric = noopMetric{}

func (noopMetric) Start(c context.Context, name string) (context.Context, Span) {
	return c, noopSpan{}
}

//------------------------------------------------------------------------------

type metric struct {
	startTime time.Time
	endTime   time.Time

	groupsMu sync.Mutex
	groups   map[string]time.Duration
}

var _ Metric = (*metric)(nil)

func (t *metric) init() {
	t.startTime = clock.Now()
}

func (t *metric) Start(c context.Context, name string) (context.Context, Span) {
	if t == nil {
		return c, noopSpan{}
	}

	parent, ok := ContextSpan(c).(*span)
	if ok {
		parent.pause()
	}
	sp := newSpan(t, name)
	sp.parent = parent

	c = context.WithValue(c, spanCtxKey, sp)
	return c, sp
}

func (t *metric) finish() {
	if t.endTime.IsZero() {
		t.endTime = clock.Now()
	}
}

func (t *metric) duration() (time.Duration, error) {
	if t.startTime.IsZero() {
		return 0, errors.New("metric.startTime is zero")
	}
	if t.endTime.IsZero() {
		return 0, errors.New("metric.endTime is zero")
	}
	return t.endTime.Sub(t.startTime), nil
}

func (t *metric) WithSpan(ctx context.Context, name string, body func(context.Context) error) error {
	ctx, span := t.Start(ctx, name)
	defer span.Finish()
	if err := body(ctx); err != nil {
		return err
	}
	return nil
}

func (t *metric) incGroup(name string, dur time.Duration) {
	if !t.endTime.IsZero() {
		return
	}

	t.groupsMu.Lock()
	if t.groups == nil {
		t.groups = make(map[string]time.Duration)
	}
	t.groups[name] += dur
	t.groupsMu.Unlock()
}

func (t *metric) flushGroups() map[string]time.Duration {
	t.groupsMu.Lock()
	groups := t.groups
	t.groups = nil
	t.groupsMu.Unlock()
	return groups
}

//------------------------------------------------------------------------------

type Span interface {
	Finish()
}

//------------------------------------------------------------------------------

type noopSpan struct{}

var _ Span = noopSpan{}

func (noopSpan) Finish() {}

//------------------------------------------------------------------------------

type span struct {
	metric *metric
	parent *span

	name  string
	start time.Time
	dur   time.Duration

	paused int32 // atomic
}

var _ Span = (*span)(nil)

func newSpan(metric *metric, name string) *span {
	return &span{
		metric: metric,
		name:   name,
		start:  clock.Now(),
	}
}

func (s *span) Finish() {
	if s.metric == nil {
		logger.Printf("gobrake: span=%q is already finished", s.name)
		return
	}
	if !s.pause() {
		return
	}

	s.metric.incGroup(s.name, s.dur)
	if s.parent != nil {
		s.parent.resume()
	}

	s.metric = nil
	s.parent = nil
}

func (s *span) pause() bool {
	if !atomic.CompareAndSwapInt32(&s.paused, 0, 1) {
		return false
	}
	s.dur += clock.Since(s.start)
	s.start = time.Time{}
	return true
}

func (s *span) resume() {
	if !atomic.CompareAndSwapInt32(&s.paused, 1, 0) {
		return
	}
	s.start = clock.Now()
}
