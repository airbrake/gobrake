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

var _ = Describe("trace", func() {
	BeforeEach(func() {
		clock = fakeClock
	})

	AfterEach(func() {
		clock = realClock
	})

	It("supports nested spans", func() {
		c := context.Background()
		var trace trace

		c, sp0 := trace.Start(c, "root")
		{
			fakeClock.Advance(time.Millisecond)
			c, sp1 := trace.Start(c, "nested1")
			{
				fakeClock.Advance(time.Millisecond)
				_, sp2 := trace.Start(c, "nested1")
				{
					fakeClock.Advance(time.Millisecond)
					sp2.Finish()
				}
				fakeClock.Advance(time.Millisecond)
				sp1.Finish()
			}
			fakeClock.Advance(time.Millisecond)
			sp0.Finish()
		}

		Expect(trace.groups["root"]).To(BeNumerically("==", 2*time.Millisecond))
		Expect(trace.groups["nested1"]).To(BeNumerically("==", 3*time.Millisecond))
		Expect(trace.groups["other"]).To(BeNumerically("==", 0))
	})

	It("supports resuming same span", func() {
		c := context.Background()
		var trace trace

		c, sp0 := trace.Start(c, "root")
		{
			fakeClock.Advance(time.Millisecond)
			c, sp1 := trace.Start(c, "nested1")
			{
				fakeClock.Advance(time.Millisecond)
				_, sp2 := trace.Start(c, "root")
				{
					fakeClock.Advance(time.Millisecond)
					sp2.Finish()
				}
				fakeClock.Advance(time.Millisecond)
				sp1.Finish()
			}
			fakeClock.Advance(time.Millisecond)
			sp0.Finish()
		}

		Expect(trace.groups["root"]).To(BeNumerically("==", 3*time.Millisecond))
		Expect(trace.groups["nested1"]).To(BeNumerically("==", 2*time.Millisecond))
		Expect(trace.groups["other"]).To(BeNumerically("==", 0))
	})

	It("supports spans on same level", func() {
		c := context.Background()
		var trace trace

		c, sp0 := trace.Start(c, "sp0")
		fakeClock.Advance(time.Millisecond)
		sp0.Finish()

		_, sp1 := trace.Start(c, "sp1")
		fakeClock.Advance(time.Millisecond)
		sp1.Finish()

		Expect(trace.groups["sp0"]).To(BeNumerically("==", 1*time.Millisecond))
		Expect(trace.groups["sp1"]).To(BeNumerically("==", 1*time.Millisecond))
		Expect(trace.groups["other"]).To(BeNumerically("==", 0))

	})
})

var _ = Describe("RouteTrace", func() {
	It("supports nil trace", func() {
		c := context.Background()
		var trace *RouteTrace
		_, span := trace.Start(c, "foo")
		span.Finish()
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
