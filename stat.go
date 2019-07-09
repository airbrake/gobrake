package gobrake

import (
	"sync"
	"time"

	tdigest "github.com/caio/go-tdigest"
)

type tdigestStat struct {
	mu      sync.Mutex
	Count   int     `json:"count"`
	Sum     float64 `json:"sum"`
	Sumsq   float64 `json:"sumsq"`
	TDigest []byte  `json:"tdigest"`
	td      *tdigest.TDigest
}

func newTDigestStat() *tdigestStat {
	return new(tdigestStat)
}

func (s *tdigestStat) Add(dur time.Duration) error {
	s.mu.Lock()
	err := s.add(dur)
	s.mu.Unlock()
	return err
}

func (s *tdigestStat) add(dur time.Duration) error {
	if s.td == nil {
		seed := time.Now().UnixNano()
		td, err := tdigest.New(
			tdigest.Compression(20), tdigest.LocalRandomNumberGenerator(seed))
		if err != nil {
			return err
		}
		s.td = td
	}

	ms := durInMs(dur)
	s.Count++
	s.Sum += ms
	s.Sumsq += ms * ms
	return s.td.Add(ms)
}

func (s *tdigestStat) Pack() error {
	err := s.td.Compress()
	if err != nil {
		return err
	}

	b, err := s.td.AsBytes()
	if err != nil {
		return err
	}
	s.TDigest = b

	return nil
}

func durInMs(dur time.Duration) float64 {
	return float64(dur) / float64(time.Millisecond)
}

type tdigestStatGroups struct {
	tdigestStat                         // total
	Groups      map[string]*tdigestStat `json:"groups"`
}

func (b *tdigestStatGroups) Add(total time.Duration, groups map[string]time.Duration) {
	b.mu.Lock()
	b.add(total, groups)
	b.mu.Unlock()
}

func (b *tdigestStatGroups) add(total time.Duration, groups map[string]time.Duration) {
	if b.Groups == nil {
		b.Groups = make(map[string]*tdigestStat)
	}

	_ = b.tdigestStat.add(total)

	var sum time.Duration
	for name, dur := range groups {
		sum += dur
		b.addGroup(name, dur)
	}

	if total > sum {
		b.addGroup("other", total-sum)
	} else {
		logger.Printf("trace total=%s <= sum=%s of groups", total, sum)
	}
}

func (b *tdigestStatGroups) addGroup(name string, dur time.Duration) {
	s, ok := b.Groups[name]
	if !ok {
		s = newTDigestStat()
		b.Groups[name] = s
	}
	_ = s.add(dur)
}

func (b *tdigestStatGroups) Pack() error {
	err := b.tdigestStat.Pack()
	if err != nil {
		return err
	}

	for _, v := range b.Groups {
		err = v.Pack()
		if err != nil {
			return err
		}
	}

	return nil
}
