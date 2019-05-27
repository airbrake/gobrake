package gobrake

import (
	"context"
)

const queueTraceCtxKey ctxKey = "ab_queue_trace"

type QueueTrace struct {
	trace
	Queue string
}

func NewQueueTrace(c context.Context, name string) (context.Context, *QueueTrace) {
	t := &QueueTrace{
		Queue: name,
	}
	t.startTime = clock.Now()
	if c != nil {
		c = context.WithValue(c, queueTraceCtxKey, t)
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
	t, _ := c.Value(queueTraceCtxKey).(*QueueTrace)
	return t
}
