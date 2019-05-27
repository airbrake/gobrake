package gobrake

import (
	"context"
)

const queueTraceCtxKey ctxKey = "ab_queue_trace"

type QueueTrace struct {
	trace
	Queue string
}

func NewQueueTrace(c context.Context, trace *QueueTrace) (context.Context, *QueueTrace) {
	trace.startTime = clock.Now()
	c = context.WithValue(c, queueTraceCtxKey, trace)
	return c, trace
}

func QueueTraceFromContext(c context.Context) *QueueTrace {
	if c == nil {
		return nil
	}
	t, _ := c.Value(queueTraceCtxKey).(*QueueTrace)
	return t
}
