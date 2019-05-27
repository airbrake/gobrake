package gobrake

import (
	"context"
)

type QueueTrace struct {
	trace
	Queue string
}

var _ Trace = (*QueueTrace)(nil)

func NewQueueTrace(c context.Context, name string) (context.Context, *QueueTrace) {
	t := &QueueTrace{
		Queue: name,
	}
	t.startTime = clock.Now()
	if c != nil {
		c = context.WithValue(c, traceCtxKey, t)
	}
	return c, t
}

func (t *QueueTrace) StartSpan(name string) {
	if t != nil {
		t.trace.StartSpan(name)
	}
}

func (t *QueueTrace) EndSpan(name string) {
	if t != nil {
		t.trace.EndSpan(name)
	}
}

func QueueTraceFromContext(c context.Context) *QueueTrace {
	if c == nil {
		return nil
	}
	t, _ := c.Value(traceCtxKey).(*QueueTrace)
	return t
}
