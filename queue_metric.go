package gobrake

import (
	"context"
)

type QueueMetric struct {
	metric
	Queue   string
	Errored bool

	root Span
}

var _ Metric = (*QueueMetric)(nil)

func NewQueueMetric(c context.Context, name string) (context.Context, *QueueMetric) {
	t := &QueueMetric{
		Queue: name,
	}
	t.metric.init()
	if c != nil {
		c = withMetric(c, t)
	}
	c, t.root = t.Start(c, "queue.handler")
	return c, t
}

func ContextQueueMetric(c context.Context) *QueueMetric {
	if c == nil {
		return nil
	}
	t, _ := c.Value(metricCtxKey).(*QueueMetric)
	return t
}

func (t *QueueMetric) Start(c context.Context, name string) (context.Context, Span) {
	if t == nil {
		return c, noopSpan{}
	}
	return t.metric.Start(c, name)
}

func (t *QueueMetric) finish() {
	t.root.Finish()
	t.metric.finish()
}
