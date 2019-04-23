package gobrake

import (
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
	It("supports nested spans", func() {
		trace := &RouteTrace{
			StartTime: clock.Now(),
		}

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
