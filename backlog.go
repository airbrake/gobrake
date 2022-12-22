package gobrake

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"sync/atomic"
	"time"
)

const (
	backlogSize        = 100
	flushBacklogPeriod = 60 * time.Second
)

type noticeBacklog struct {
	opt       *NotifierOptions
	notices   []Notice
	maxLength int
}

type apmBacklog struct {
	opt             *NotifierOptions
	routeStats      []routesOut
	routeBreakdowns []breakdownsOut
	queries         []queriesOut
	queues          []queuesOut
}

var (
	nb              *noticeBacklog
	ab              *apmBacklog
	apmBacklogCount int32
)

// Purge reset the backlog APM count to 0.
func purge() {
	atomic.StoreInt32(&apmBacklogCount, 0)
}

// inc increase the backlog APM count by 1.
func inc() {
	atomic.StoreInt32(&apmBacklogCount, getCount()+1)
}

// getCount returns the backlog APM count.
func getCount() int32 {
	return atomic.LoadInt32(&apmBacklogCount)
}

// newBacklog creates a new backlog for notices and APM stats.
func newBacklog(opt *NotifierOptions) {
	nb = &noticeBacklog{
		maxLength: backlogSize,
		opt:       opt,
	}
	ab = &apmBacklog{
		opt: opt,
	}
}

// setNoticeBacklog sets new backlog notice.
func setNoticeBacklog(notice *Notice) {
	if nb.opt.DisableBacklog {
		return
	}
	if len(nb.notices) < nb.maxLength {
		nb.notices = append(nb.notices, *notice)
	}
	for {
		<-time.After(flushBacklogPeriod)
		nb.flushNoticeBacklog()
	}
}

// flushNoticeBacklog sends the backlog notice after the backlog period is over.
func (nb *noticeBacklog) flushNoticeBacklog() {
	buf := buffers.Get().(*bytes.Buffer)

	for _, notice := range nb.notices {
		err := json.NewEncoder(buf).Encode(notice)
		if err != nil {
			logger.Printf("Backlog notice failed = %s", err)
			continue
		}

		req, err := http.NewRequest(
			http.MethodPost,
			fmt.Sprintf("%s/api/v3/projects/%d/notices",
				nb.opt.Host, nb.opt.ProjectId),
			buf,
		)
		if err != nil {
			logger.Printf("Backlog notice failed = %s", err)
			continue
		}

		req.Header.Set("Authorization", "Bearer "+nb.opt.ProjectKey)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("User-Agent", userAgent)

		resp, err := nb.opt.HTTPClient.Do(req)
		if err != nil {
			logger.Printf("Backlog notice failed = %s", err)
			continue
		}
		defer resp.Body.Close()

		if resp.StatusCode > 400 {
			logger.Printf("Backlog notice failed = %q", resp.Status)
		}

		buf.Reset()
	}

	nb.notices = nil
}

// setRouteStatBacklog sets new backlog route stat.
func setRouteStatBacklog(routeStat routesOut) {
	if ab.opt.DisableBacklog {
		return
	}
	if getCount() < backlogSize {
		ab.routeStats = append(ab.routeStats, routeStat)
		inc()
	}
	for {
		<-time.After(flushBacklogPeriod)
		ab.flushRouteStatBacklog()
	}
}

// flushRouteStatBacklog sends the backlog route stats after the backlog period is over.
func (ab *apmBacklog) flushRouteStatBacklog() {
	buf := buffers.Get().(*bytes.Buffer)

	for _, routeStat := range ab.routeStats {
		err := json.NewEncoder(buf).Encode(routeStat)
		if err != nil {
			logger.Printf("Backlog route stat failed = %s", err)
			continue
		}

		req, err := http.NewRequest(
			http.MethodPut,
			fmt.Sprintf("%s/api/v5/projects/%d/routes-stats",
				ab.opt.APMHost, ab.opt.ProjectId),
			buf,
		)
		if err != nil {
			logger.Printf("Backlog route stat failed = %s", err)
			continue
		}

		req.Header.Set("Authorization", "Bearer "+ab.opt.ProjectKey)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("User-Agent", userAgent)

		resp, err := ab.opt.HTTPClient.Do(req)
		if err != nil {
			logger.Printf("Backlog route stat failed = %s", err)
			continue
		}
		defer resp.Body.Close()

		if resp.StatusCode > 400 {
			logger.Printf("Backlog route stat failed = %q", resp.Status)
		}

		buf.Reset()
	}

	ab.routeStats = nil
	purge()
}

// setRouteBreakdownBacklog sets new backlog route breakdown.
func setRouteBreakdownBacklog(routeBreakdown breakdownsOut) {
	if ab.opt.DisableBacklog {
		return
	}
	if getCount() < backlogSize {
		ab.routeBreakdowns = append(ab.routeBreakdowns, routeBreakdown)
		inc()
	}
	for {
		<-time.After(flushBacklogPeriod)
		ab.flushRouteBreakdownBacklog()
	}
}

// flushBacklogRouteBreakdown sends the backlog route breakdowns after the backlog period is over.
func (ab *apmBacklog) flushRouteBreakdownBacklog() {
	buf := buffers.Get().(*bytes.Buffer)

	for _, routeBreakdown := range ab.routeBreakdowns {
		err := json.NewEncoder(buf).Encode(routeBreakdown)
		if err != nil {
			logger.Printf("Backlog route stat failed = %s", err)
			continue
		}

		req, err := http.NewRequest(
			http.MethodPut,
			fmt.Sprintf("%s/api/v5/projects/%d/routes-breakdowns",
				ab.opt.APMHost, ab.opt.ProjectId),
			buf,
		)
		if err != nil {
			logger.Printf("Backlog route stat failed = %s", err)
			continue
		}

		req.Header.Set("Authorization", "Bearer "+ab.opt.ProjectKey)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("User-Agent", userAgent)

		resp, err := ab.opt.HTTPClient.Do(req)
		if err != nil {
			logger.Printf("Backlog route stat failed = %s", err)
			continue
		}
		defer resp.Body.Close()

		if resp.StatusCode > 400 {
			logger.Printf("Backlog route stat failed = %q", resp.Status)
		}

		buf.Reset()
	}

	ab.routeBreakdowns = nil
}

// setQueryBacklog sets new backlog query.
func setQueryBacklog(query queriesOut) {
	if ab.opt.DisableBacklog {
		return
	}
	if getCount() < backlogSize {
		ab.queries = append(ab.queries, query)
		inc()
	}
	for {
		<-time.After(flushBacklogPeriod)
		ab.flushQueryBacklog()
	}
}

// flushQueryBacklog sends the backlog query after the backlog period is over.
func (ab *apmBacklog) flushQueryBacklog() {
	buf := buffers.Get().(*bytes.Buffer)

	for _, query := range ab.queries {
		err := json.NewEncoder(buf).Encode(query)
		if err != nil {
			logger.Printf("Backlog query failed = %s", err)
			continue
		}

		req, err := http.NewRequest(
			http.MethodPut,
			fmt.Sprintf("%s/api/v5/projects/%d/queries-stats",
				ab.opt.APMHost, ab.opt.ProjectId),
			buf,
		)
		if err != nil {
			logger.Printf("Backlog query failed = %s", err)
			continue
		}

		req.Header.Set("Authorization", "Bearer "+ab.opt.ProjectKey)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("User-Agent", userAgent)

		resp, err := ab.opt.HTTPClient.Do(req)
		if err != nil {
			logger.Printf("Backlog query failed = %s", err)
			continue
		}
		defer resp.Body.Close()

		if resp.StatusCode > 400 {
			logger.Printf("Backlog query failed = %q", resp.Status)
		}

		buf.Reset()
	}

	ab.queries = nil
}

// setQueueBacklog sets new queue backlog.
func setQueueBacklog(queue queuesOut) {
	if ab.opt.DisableBacklog {
		return
	}
	if getCount() < backlogSize {
		ab.queues = append(ab.queues, queue)
		inc()
	}
	for {
		<-time.After(flushBacklogPeriod)
		ab.flushQueueBacklog()
	}
}

// flushQueueBacklog sends the queue backlog after the backlog period is over.
func (ab *apmBacklog) flushQueueBacklog() {
	buf := buffers.Get().(*bytes.Buffer)

	for _, queue := range ab.queues {
		err := json.NewEncoder(buf).Encode(queue)
		if err != nil {
			logger.Printf("Backlog queue failed = %s", err)
			continue
		}

		req, err := http.NewRequest(
			http.MethodPut,
			fmt.Sprintf("%s/api/v5/projects/%d/queues-stats",
				ab.opt.APMHost, ab.opt.ProjectId),
			buf,
		)
		if err != nil {
			logger.Printf("Backlog queue failed = %s", err)
			continue
		}

		req.Header.Set("Authorization", "Bearer "+ab.opt.ProjectKey)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("User-Agent", userAgent)

		resp, err := ab.opt.HTTPClient.Do(req)
		if err != nil {
			logger.Printf("Backlog queue failed = %s", err)
			continue
		}
		defer resp.Body.Close()

		if resp.StatusCode > 400 {
			logger.Printf("Backlog queue failed = %q", resp.Status)
		}

		buf.Reset()
	}

	ab.queues = nil
}
