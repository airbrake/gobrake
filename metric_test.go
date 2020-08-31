package gobrake

import (
	"bytes"
	"context"
	"log"
	"net/http"
	"net/http/httptest"
	"sync"
	"time"

	"github.com/jonboulle/clockwork"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var (
	realClock = clockwork.NewRealClock()
	fakeClock = clockwork.NewFakeClock()
)

var _ = Describe("metric with real clock", func() {
	It("supports measuring spans in goroutines", func() {
		c := context.Background()

		var metric metric
		metric.init()

		c, sp0 := metric.Start(c, "sp0")
		time.Sleep(100 * time.Millisecond)

		var wg sync.WaitGroup
		for i := 0; i < 2; i++ {
			wg.Add(1)
			go func(c context.Context) {
				defer wg.Done()

				_, sp1 := metric.Start(c, "sp1")
				time.Sleep(100 * time.Millisecond)
				sp1.Finish()
			}(c)
		}
		wg.Wait()

		sp0.Finish()
		metric.finish()

		Expect(metric.groups["sp0"]).To(BeNumerically("==", 100*time.Millisecond, 20*time.Millisecond))
		Expect(metric.groups["sp1"]).To(BeNumerically("==", 200*time.Millisecond, 20*time.Millisecond))
		Expect(metric.duration()).To(BeNumerically("==", 200*time.Millisecond, 20*time.Millisecond))
	})
})

var _ = Describe("metric with fake clock", func() {
	BeforeEach(func() {
		clock = fakeClock
	})

	AfterEach(func() {
		clock = realClock
	})

	It("supports nested spans", func() {
		c := context.Background()
		var metric metric

		c, sp0 := metric.Start(c, "root")
		{
			fakeClock.Advance(time.Millisecond)
			c, sp1 := metric.Start(c, "nested1")
			{
				fakeClock.Advance(time.Millisecond)
				_, sp2 := metric.Start(c, "nested1")
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

		Expect(metric.groups["root"]).To(BeNumerically("==", 2*time.Millisecond))
		Expect(metric.groups["nested1"]).To(BeNumerically("==", 3*time.Millisecond))
	})

	It("supports resuming same span", func() {
		c := context.Background()
		var metric metric

		c, sp0 := metric.Start(c, "root")
		{
			fakeClock.Advance(time.Millisecond)
			c, sp1 := metric.Start(c, "nested1")
			{
				fakeClock.Advance(time.Millisecond)
				_, sp2 := metric.Start(c, "root")
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

		Expect(metric.groups["root"]).To(BeNumerically("==", 3*time.Millisecond))
		Expect(metric.groups["nested1"]).To(BeNumerically("==", 2*time.Millisecond))
	})

	It("supports spans on same level", func() {
		c := context.Background()
		var metric metric

		c, sp0 := metric.Start(c, "sp0")
		fakeClock.Advance(time.Millisecond)
		sp0.Finish()

		_, sp1 := metric.Start(c, "sp1")
		fakeClock.Advance(time.Millisecond)
		sp1.Finish()

		Expect(metric.groups["sp0"]).To(BeNumerically("==", 1*time.Millisecond))
		Expect(metric.groups["sp1"]).To(BeNumerically("==", 1*time.Millisecond))
	})
})

var _ = Describe("httpmetric", func() {
	var server *httptest.Server

	BeforeEach(func() {
		handler := func(w http.ResponseWriter, req *http.Request) {
			w.WriteHeader(http.StatusNotFound)
			_, err := w.Write([]byte(""))
			Expect(err).To(BeNil())
		}
		server = httptest.NewServer(http.HandlerFunc(handler))
	})

	It("measures timing until first response byte", func() {
		c := context.Background()
		c, metric := NewRouteMetric(c, "GET", "/api/v1/projects/:projectId")

		req, _ := http.NewRequest("GET", server.URL, nil)
		req = req.WithContext(c)

		_, err := http.DefaultClient.Do(req)
		Expect(err).NotTo(HaveOccurred())

		Expect(metric.groups).To(HaveLen(1))
		Expect(metric.groups["http.client"]).NotTo(BeZero())
	})

	It("doesn't attempt to finish the same span multiple times", func() {
		origLogger := GetLogger()
		defer func() {
			SetLogger(origLogger)
		}()

		buf := new(bytes.Buffer)
		l := log.New(buf, "", 0)
		SetLogger(l)

		c := context.Background()
		c, _ = NewQueueMetric(c, "send-emails")

		wg := &sync.WaitGroup{}
		for i := 0; i < 3; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				req, _ := http.NewRequest("GET", server.URL, nil)
				req = req.WithContext(c)
				_, err := http.DefaultClient.Do(req)
				Expect(err).NotTo(HaveOccurred())
			}()
		}

		wg.Wait()

		Expect(buf.String()).To(BeEmpty())
	})
})

var _ = Describe("RouteMetric", func() {
	BeforeEach(func() {
		clock = fakeClock
	})

	AfterEach(func() {
		clock = realClock
	})

	It("supports nil metric", func() {
		c := context.Background()
		var metric *RouteMetric
		_, span := metric.Start(c, "foo")
		span.Finish()
	})

	It("automatically starts http.handler span", func() {
		c := context.Background()

		c, metric := NewRouteMetric(c, "GET", "/foo")
		fakeClock.Advance(time.Millisecond)

		_, sp0 := metric.Start(c, "sp0")
		fakeClock.Advance(time.Millisecond)
		sp0.Finish()

		metric.finish()

		Expect(metric.groups["http.handler"]).To(BeNumerically("==", 1*time.Millisecond))
		Expect(metric.groups["sp0"]).To(BeNumerically("==", 1*time.Millisecond))
		Expect(metric.duration()).To(BeNumerically("==", 2*time.Millisecond))
	})
})

var _ = Describe("QueueMetric", func() {
	BeforeEach(func() {
		clock = fakeClock
	})

	AfterEach(func() {
		clock = realClock
	})

	It("supports nil metric", func() {
		c := context.Background()
		var metric *QueueMetric
		_, span := metric.Start(c, "foo")
		span.Finish()
	})

	It("automatically starts queue.handler span", func() {
		c := context.Background()

		c, metric := NewQueueMetric(c, "send-emails")
		fakeClock.Advance(time.Millisecond)

		_, sp0 := metric.Start(c, "sp0")
		fakeClock.Advance(time.Millisecond)
		sp0.Finish()

		metric.finish()

		Expect(metric.groups["queue.handler"]).To(BeNumerically("==", 1*time.Millisecond))
		Expect(metric.groups["sp0"]).To(BeNumerically("==", 1*time.Millisecond))
		Expect(metric.duration()).To(BeNumerically("==", 2*time.Millisecond))
	})
})
