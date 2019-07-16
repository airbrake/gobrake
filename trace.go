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

const traceCtxKey ctxKey = "ab_trace"
const spanCtxKey ctxKey = "ab_span"

type Trace interface {
	Start(c context.Context, name string) (context.Context, Span)
}

func withTrace(c context.Context, t Trace) context.Context {
	c = context.WithValue(c, traceCtxKey, t)

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

func ContextTrace(c context.Context) Trace {
	if c == nil {
		return noopTrace{}
	}
	if t, ok := c.Value(traceCtxKey).(Trace); ok {
		return t
	}
	return noopTrace{}
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

type noopTrace struct{}

var _ Trace = noopTrace{}

func (noopTrace) Start(c context.Context, name string) (context.Context, Span) {
	return c, noopSpan{}
}

//------------------------------------------------------------------------------

type trace struct {
	startTime time.Time
	endTime   time.Time

	groupsMu sync.Mutex
	groups   map[string]time.Duration
}

var _ Trace = (*trace)(nil)

func (t *trace) init() {
	t.startTime = clock.Now()
}

func (t *trace) Start(c context.Context, name string) (context.Context, Span) {
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

func (t *trace) finish() {
	if t.endTime.IsZero() {
		t.endTime = clock.Now()
	}
}

func (t *trace) duration() (time.Duration, error) {
	if t.startTime.IsZero() {
		return 0, errors.New("trace.startTime is zero")
	}
	if t.endTime.IsZero() {
		return 0, errors.New("trace.endTime is zero")
	}
	return t.endTime.Sub(t.startTime), nil
}

func (t *trace) WithSpan(ctx context.Context, name string, body func(context.Context) error) error {
	ctx, span := t.Start(ctx, name)
	defer span.Finish()
	if err := body(ctx); err != nil {
		return err
	}
	return nil
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
	Finish()
}

//------------------------------------------------------------------------------

type noopSpan struct{}

var _ Span = noopSpan{}

func (noopSpan) Finish() {}

//------------------------------------------------------------------------------

type span struct {
	trace  *trace
	parent *span

	name  string
	start time.Time
	dur   time.Duration

	paused int32 // atomic
}

var _ Span = (*span)(nil)

func newSpan(trace *trace, name string) *span {
	return &span{
		trace: trace,
		name:  name,
		start: clock.Now(),
	}
}

func (s *span) Finish() {
	if s.trace == nil {
		logger.Printf("gobrake: span=%q is already finished", s.name)
		return
	}
	if !s.pause() {
		return
	}

	s.trace.incGroup(s.name, s.dur)
	if s.parent != nil {
		s.parent.resume()
	}

	s.trace = nil
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
