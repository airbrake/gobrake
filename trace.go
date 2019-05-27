package gobrake

import (
	"sync"
	"time"
)

type trace struct {
	startTime time.Time
	endTime   time.Time

	spansMu  sync.Mutex
	spans    map[string]*span
	currSpan *span

	groupsMu sync.Mutex
	groups   map[string]time.Duration
}

func (t *trace) end() {
	if t.endTime.IsZero() {
		t.endTime = clock.Now()
	}
}

func (t *trace) StartSpan(name string) {
	if t == nil {
		return
	}

	t.spansMu.Lock()
	defer t.spansMu.Unlock()

	if t.spans == nil {
		t.spans = make(map[string]*span)
	}

	if t.currSpan != nil {
		if t.currSpan.name == name {
			t.currSpan.level++
			return
		}

		t.currSpan.pause()
	}

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

func (t *trace) EndSpan(name string) {
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

func (t *trace) endSpan(span *span) bool {
	if span.level > 0 {
		span.level--
		return false
	}
	span.End()
	delete(t.spans, span.name)
	return true
}

func (t *trace) incGroup(name string, dur time.Duration) {
	t.groupsMu.Lock()
	if !t.endTime.IsZero() {
		return
	}
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

type Span interface {
	End()
}

type span struct {
	trace  *trace
	parent *span

	name  string
	start time.Time
	dur   time.Duration
	level int
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
