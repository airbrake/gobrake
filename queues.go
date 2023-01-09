package gobrake

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sync"
	"time"
)

type queueKey struct {
	Queue string    `json:"queue"`
	Time  time.Time `json:"time"`
}

type queueBreakdown struct {
	queueKey
	tdigestStatGroups
	ErrorCount int `json:"errorCount"`
}

func (b *queueBreakdown) Add(total time.Duration, groups map[string]time.Duration, errored bool) {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.tdigestStatGroups.add(total, groups)
	if errored {
		b.ErrorCount++
	}
}

type queueStats struct {
	opt        *NotifierOptions
	flushTimer *time.Timer
	addWG      *sync.WaitGroup

	mu sync.Mutex
	m  map[queueKey]*queueBreakdown
}

func newQueueStats(opt *NotifierOptions) *queueStats {
	return &queueStats{
		opt: opt,
	}
}

func (s *queueStats) init() {
	if s.flushTimer == nil {
		s.flushTimer = time.AfterFunc(flushPeriod, s.flush)
		s.addWG = new(sync.WaitGroup)
		s.m = make(map[queueKey]*queueBreakdown)
	}
}

func (s *queueStats) flush() {
	s.mu.Lock()

	s.flushTimer = nil
	addWG := s.addWG
	s.addWG = nil
	m := s.m
	s.m = nil

	s.mu.Unlock()

	addWG.Wait()
	err := s.send(m)
	if err != nil {
		logger.Printf("queueStats.send failed: %s", err)
	}
}

type queuesOut struct {
	Env    string            `json:"environment"`
	Queues []*queueBreakdown `json:"queues"`
}

func (s *queueStats) send(m map[queueKey]*queueBreakdown) error {
	var queues []*queueBreakdown
	for _, v := range m {
		err := v.Pack()
		if err != nil {
			return err
		}
		queues = append(queues, v)
	}

	buf := buffers.Get().(*bytes.Buffer)
	defer buffers.Put(buf)
	buf.Reset()

	out := queuesOut{
		Env:    s.opt.Environment,
		Queues: queues,
	}
	err := json.NewEncoder(buf).Encode(&out)
	if err != nil {
		return err
	}

	req, err := http.NewRequest(
		http.MethodPut,
		fmt.Sprintf("%s/api/v5/projects/%d/queues-stats",
			s.opt.APMHost, s.opt.ProjectId),
		buf,
	)
	if err != nil {
		return err
	}

	req.Header.Set("Authorization", "Bearer "+s.opt.ProjectKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", userAgent)
	resp, err := s.opt.HTTPClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	buf.Reset()
	_, err = buf.ReadFrom(resp.Body)
	if err != nil {
		return err
	}

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return nil
	}

	switch resp.StatusCode {
	case http.StatusUnauthorized:
		return errUnauthorized
	case http.StatusBadRequest, http.StatusTooManyRequests:
		var sendResp sendResponse
		err = json.NewDecoder(buf).Decode(&sendResp)
		if err != nil {
			return err
		}
		return errors.New(sendResp.Message)
	case 404, 408, 409, 410, 500, 502, 504:
		setQueueBacklog(out)
	}

	err = fmt.Errorf("got unexpected response status=%q", resp.Status)
	return err
}

func (s *queueStats) Notify(c context.Context, metric *QueueMetric) error {
	if s.opt.DisableAPM {
		return fmt.Errorf(
			"APM is disabled, queue is not sent: %s", metric.Queue,
		)
	}

	metric.finish()

	total, err := metric.duration()
	if err != nil {
		return err
	}

	key := queueKey{
		Queue: metric.Queue,
		Time:  metric.startTime.UTC().Truncate(time.Minute),
	}

	s.mu.Lock()
	s.init()
	b, ok := s.m[key]
	if !ok {
		b = &queueBreakdown{
			queueKey: key,
		}
		s.m[key] = b
	}
	addWG := s.addWG
	s.addWG.Add(1)
	s.mu.Unlock()

	groups := metric.flushGroups()
	b.Add(total, groups, metric.Errored)
	addWG.Done()

	return nil
}
