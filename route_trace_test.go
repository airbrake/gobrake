package gobrake

import (
	"context"
	"net/http"
	"time"

	"github.com/jonboulle/clockwork"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var (
	realClock = clockwork.NewRealClock()
	fakeClock = clockwork.NewFakeClock()
)

var _ = Describe("RouteTrace", func() {
	BeforeEach(func() {
		clock = fakeClock
	})

	AfterEach(func() {
		clock = realClock
	})

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
})

var _ = Describe("httptrace", func() {
	It("measures timing until first response byte", func() {
		c := context.Background()
		c, trace := NewRouteTrace(c, "GET", "/api/v1/projects/:projectId")

		req, _ := http.NewRequest("GET", "https://www.google.com/", nil)
		req = req.WithContext(c)

		_, err := http.DefaultClient.Do(req)
		Expect(err).NotTo(HaveOccurred())

		Expect(trace.groups).To(HaveLen(1))
		Expect(trace.groups["http.client"]).NotTo(BeZero())
	})
})
