package gobrake

import (
	"context"
)

type QueueTrace struct {
	trace
	Queue   string
	Errored bool

	root Span
}

var _ Trace = (*QueueTrace)(nil)

func NewQueueTrace(c context.Context, name string) (context.Context, *QueueTrace) {
	t := &QueueTrace{
		Queue: name,
	}
	t.init()
	if c != nil {
		c = withTrace(c, t)
	}
	c, t.root = t.Start(c, "queue.handler")
	return c, t
}

func ContextQueueTrace(c context.Context) *QueueTrace {
	if c == nil {
		return nil
	}
	t, _ := c.Value(traceCtxKey).(*QueueTrace)
	return t
}

func (t *QueueTrace) Start(c context.Context, name string) (context.Context, Span) {
	if t == nil {
		return c, noopSpan{}
	}
	return t.trace.Start(c, name)
}

func (t *QueueTrace) finish() {
	t.root.Finish()
	t.trace.finish()
}
