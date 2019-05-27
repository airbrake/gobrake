package gobrake

import (
	"context"
	"strings"
)

type RouteTrace struct {
	trace
	Method      string
	Route       string
	StatusCode  int
	ContentType string
}

var _ Trace = (*RouteTrace)(nil)

func NewRouteTrace(c context.Context, method, route string) (context.Context, *RouteTrace) {
	t := &RouteTrace{
		Method: method,
		Route:  route,
	}
	t.startTime = clock.Now()
	if c != nil {
		c = context.WithValue(c, traceCtxKey, t)
	}
	return c, t
}

func RouteTraceFromContext(c context.Context) *RouteTrace {
	if c == nil {
		return nil
	}
	t, _ := c.Value(traceCtxKey).(*RouteTrace)
	return t
}

func (t *RouteTrace) StartSpan(name string) {
	if t != nil {
		t.trace.StartSpan(name)
	}
}

func (t *RouteTrace) EndSpan(name string) {
	if t != nil {
		t.trace.EndSpan(name)
	}
}

func (t *RouteTrace) respType() string {
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
