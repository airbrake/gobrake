package gobrake

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"
)

type QueueInfo struct {
	Queue string
	Start time.Time
	End   time.Time
}

type queueKey struct {
	Queue string    `json:"queue"`
	Time  time.Time `json:"time"`
}

type queueKeyStat struct {
	queueKey
	*routeStat
}

type queueStats struct {
	opt    *NotifierOptions
	apiURL string

	flushTimer *time.Timer
	addWG      *sync.WaitGroup

	mu sync.Mutex
	m  map[queueKey]*routeStat
}

func newQueueStats(opt *NotifierOptions) *queueStats {
	return &queueStats{
		opt: opt,
		apiURL: fmt.Sprintf("%s/api/v5/projects/%d/queues-stats",
			opt.Host, opt.ProjectId),
	}
}

func (s *queueStats) init() {
	if s.flushTimer == nil {
		s.flushTimer = time.AfterFunc(flushPeriod, s.flush)
		s.addWG = new(sync.WaitGroup)
		s.m = make(map[queueKey]*routeStat)
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
	Env    string         `json:"environment"`
	Queues []queueKeyStat `json:"queues"`
}

func (s *queueStats) send(m map[queueKey]*routeStat) error {
	var queues []queueKeyStat
	for k, v := range m {
		err := v.td.Compress()
		if err != nil {
			return err
		}

		b, err := v.td.AsBytes()
		if err != nil {
			return err
		}
		v.TDigest = b

		queues = append(queues, queueKeyStat{
			queueKey:  k,
			routeStat: v,
		})
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

	req, err := http.NewRequest("PUT", s.apiURL, buf)
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
	case httpStatusTooManyRequests:
		return errIPRateLimited
	}

	err = fmt.Errorf("got unexpected response status=%q", resp.Status)
	return err
}

func (s *queueStats) Notify(c context.Context, q *QueueInfo) error {
	key := queueKey{
		Queue: q.Queue,
		Time:  q.Start.UTC().Truncate(time.Minute),
	}

	s.mu.Lock()
	s.init()
	stat, ok := s.m[key]
	if !ok {
		stat = &routeStat{}
		s.m[key] = stat
	}
	addWG := s.addWG
	s.addWG.Add(1)
	s.mu.Unlock()

	dur := q.End.Sub(q.Start)

	stat.mu.Lock()
	err := stat.Add(dur)
	addWG.Done()
	stat.mu.Unlock()

	return err
}
