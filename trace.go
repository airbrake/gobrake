package gobrake

import (
	"context"
	"errors"
	"net/http/httptrace"
	"sync"
	"time"
)

type ctxKey string

const traceCtxKey ctxKey = "ab_trace"

type Trace interface {
	StartSpan(name string) Span
}

func withTrace(c context.Context, t Trace) context.Context {
	c = context.WithValue(c, traceCtxKey, t)

	var span Span
	clientTrace := &httptrace.ClientTrace{
		GetConn: func(hostPort string) {
			span = t.StartSpan("http.client")
		},
		GotFirstResponseByte: func() {
			span.End()
		},
	}
	c = httptrace.WithClientTrace(c, clientTrace)

	return c
}

func ContextTrace(c context.Context) Trace {
	if c == nil {
		return noopTrace{}
	}
	t, ok := c.Value(traceCtxKey).(Trace)
	if !ok {
		return noopTrace{}
	}
	return t
}

//------------------------------------------------------------------------------

type noopTrace struct{}

var _ Trace = noopTrace{}

func (noopTrace) StartSpan(name string) Span {
	return noopSpan{}
}

//------------------------------------------------------------------------------

type trace struct {
	startTime time.Time
	endTime   time.Time

	spansMu  sync.Mutex
	currSpan *span

	groupsMu sync.Mutex
	groups   map[string]time.Duration
}

var _ Trace = (*trace)(nil)

func (t *trace) end() {
	if t.endTime.IsZero() {
		t.endTime = clock.Now()
	}
}

func (t *trace) Duration() (time.Duration, error) {
	if t.startTime.IsZero() {
		return 0, errors.New("trace.startTime is zero")
	}
	if t.endTime.IsZero() {
		return 0, errors.New("trace.endTime is zero")
	}
	return t.endTime.Sub(t.startTime), nil
}

func (t *trace) StartSpan(name string) Span {
	if t == nil {
		return noopSpan{}
	}

	t.spansMu.Lock()
	defer t.spansMu.Unlock()

	span := newSpan(t, name)
	if t.currSpan != nil {
		t.currSpan.pause()
		span.parent = t.currSpan
	}
	t.currSpan = span

	return span
}

func (t *trace) incGroup(name string, dur time.Duration) {
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

func (t *trace) flushGroups() map[string]time.Duration {
	t.groupsMu.Lock()
	groups := t.groups
	t.groups = nil
	t.groupsMu.Unlock()
	return groups
}

//------------------------------------------------------------------------------

type Span interface {
	End()
}

//------------------------------------------------------------------------------

type noopSpan struct{}

var _ Span = noopSpan{}

func (noopSpan) End() {}

//------------------------------------------------------------------------------

type span struct {
	trace  *trace
	parent *span

	name  string
	start time.Time
	dur   time.Duration
}

var _ Span = (*span)(nil)

func newSpan(trace *trace, name string) *span {
	return &span{
		trace: trace,
		name:  name,
		start: clock.Now(),
	}
}

func (s *span) End() {
	if s.trace == nil {
		logger.Printf("gobrake: span=%q is already ended", s.name)
		return
	}

	s.dur += clock.Since(s.start)
	s.trace.incGroup(s.name, s.dur)
	if s.parent != nil {
		s.parent.resume()
	}

	s.trace = nil
	s.parent = nil
}

func (s *span) pause() {
	if s.paused() {
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
