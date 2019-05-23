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

type QueryInfo struct {
	Method string
	Route  string
	Query  string
	Func   string
	File   string
	Line   int
	Start  time.Time
	End    time.Time
}

type queryKey struct {
	Method string    `json:"method"`
	Route  string    `json:"route"`
	Query  string    `json:"query"`
	Func   string    `json:"function"`
	File   string    `json:"file"`
	Line   int       `json:"line"`
	Time   time.Time `json:"time"`
}

type queryKeyStat struct {
	queryKey
	*routeStat
}

type queryStats struct {
	opt    *NotifierOptions
	apiURL string

	flushTimer *time.Timer
	addWG      *sync.WaitGroup

	mu sync.Mutex
	m  map[queryKey]*routeStat
}

func newQueryStats(opt *NotifierOptions) *queryStats {
	return &queryStats{
		opt: opt,
		apiURL: fmt.Sprintf("%s/api/v5/projects/%d/queries-stats",
			opt.Host, opt.ProjectId),
	}
}

func (s *queryStats) init() {
	if s.flushTimer == nil {
		s.flushTimer = time.AfterFunc(flushPeriod, s.flush)
		s.addWG = new(sync.WaitGroup)
		s.m = make(map[queryKey]*routeStat)
	}
}

func (s *queryStats) flush() {
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
		logger.Printf("queryStats.send failed: %s", err)
	}
}

type queriesOut struct {
	Env     string         `json:"environment"`
	Queries []queryKeyStat `json:"queries"`
}

func (s *queryStats) send(m map[queryKey]*routeStat) error {
	var queries []queryKeyStat
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

		queries = append(queries, queryKeyStat{
			queryKey:  k,
			routeStat: v,
		})
	}

	buf := buffers.Get().(*bytes.Buffer)
	defer buffers.Put(buf)
	buf.Reset()

	out := queriesOut{
		Env:     s.opt.Environment,
		Queries: queries,
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
	}

	err = fmt.Errorf("got unexpected response status=%q", resp.Status)
	return err
}

func (s *queryStats) Notify(c context.Context, q *QueryInfo) error {
	key := queryKey{
		Method: q.Method,
		Route:  q.Route,
		Query:  q.Query,
		Func:   q.Func,
		File:   q.File,
		Line:   q.Line,
		Time:   q.Start.UTC().Truncate(time.Minute),
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
