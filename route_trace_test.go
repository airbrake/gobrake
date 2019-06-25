package gobrake

import (
	"context"
	"net/http"
	"net/http/httptrace"
	"time"

	"github.com/jonboulle/clockwork"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var fakeClock = clockwork.NewFakeClock()

func init() {
	clock = fakeClock
}

var _ = Describe("RouteTrace", func() {
	It("supports nil trace", func() {
		var trace *RouteTrace
		trace.StartSpan("foo")
		trace.EndSpan("bar")
	})

	It("supports nested spans", func() {
		_, trace := NewRouteTrace(nil, "GET", "/some")

		trace.StartSpan("root")
		fakeClock.Advance(time.Millisecond)

		trace.StartSpan("nested1")
		fakeClock.Advance(time.Millisecond)

		trace.StartSpan("nested1")
		fakeClock.Advance(time.Millisecond)

		trace.EndSpan("nested1")

		fakeClock.Advance(time.Millisecond)
		trace.EndSpan("nested1")

		fakeClock.Advance(time.Millisecond)
		trace.EndSpan("root")

		Expect(trace.groups["root"]).To(BeNumerically("==", 2*time.Millisecond))
		Expect(trace.groups["nested1"]).To(BeNumerically("==", 3*time.Millisecond))
		Expect(trace.groups["other"]).To(BeNumerically("==", 0))
	})

	It("adds request tracing", func() {
		c := context.Background()
		c, trace := NewRouteTrace(c, "GET", "https://google.com")

		req, _ := http.NewRequest(trace.Method, trace.Route, nil)
		req = req.WithContext(c)

		clientTrace := &httptrace.ClientTrace{
			ConnectDone: func(network, addr string, err error) {
				fakeClock.Advance(time.Millisecond)
			},
		}
		c = httptrace.WithClientTrace(req.Context(), clientTrace)
		req = req.WithContext(c)

		client := &http.Client{}
		_, err := client.Do(req)
		Expect(err).NotTo(HaveOccurred())

		Expect(trace.groups).To(HaveLen(1))
		Expect(trace.groups["GET:https://google.com"]).To(Equal(2 * time.Millisecond))
	})
})
