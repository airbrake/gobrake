package gobrake

import (
	"context"
	"strings"
)

type RouteMetric struct {
	metric
	Method      string
	Route       string
	StatusCode  int
	ContentType string

	root Span
}

var _ Metric = (*RouteMetric)(nil)

func NewRouteMetric(c context.Context, method, route string) (context.Context, *RouteMetric) {
	t := &RouteMetric{
		Method: method,
		Route:  route,
	}
	t.metric.init()
	if c != nil {
		c = withMetric(c, t)
	}
	c, t.root = t.Start(c, "http.handler")
	return c, t
}

func ContextRouteMetric(c context.Context) *RouteMetric {
	if c == nil {
		return nil
	}
	t, _ := c.Value(metricCtxKey).(*RouteMetric)
	return t
}

func (t *RouteMetric) Start(c context.Context, name string) (context.Context, Span) {
	if t == nil {
		return c, noopSpan{}
	}
	return t.metric.Start(c, name)
}

func (t *RouteMetric) finish() {
	t.root.Finish()
	t.metric.finish()
}

func (t *RouteMetric) respType() string {
	if t.StatusCode >= 500 {
		return "5xx"
	}
	if t.StatusCode >= 400 {
		return "4xx"
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
