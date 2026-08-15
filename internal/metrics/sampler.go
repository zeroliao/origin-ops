package metrics

import (
	"context"
	"log/slog"
	"sync"
	"time"
)

type Sampler struct {
	collector Collector
	store     Appender
	interval  time.Duration

	mu          sync.RWMutex
	latest      Snapshot
	hasLatest   bool
	lastError   string
	lastFailure time.Time
}

func NewSampler(collector Collector, store Appender, interval time.Duration) *Sampler {
	return &Sampler{collector: collector, store: store, interval: interval}
}

func (s *Sampler) Run(ctx context.Context) {
	s.sample()
	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.sample()
		}
	}
}

func (s *Sampler) sample() {
	snapshot, err := s.collector.Collect()
	if err == nil {
		err = s.store.Append(snapshot)
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if err != nil {
		s.lastError = err.Error()
		s.lastFailure = time.Now().UTC()
		slog.Warn("metric sample failed", "error", err)
		return
	}

	s.latest = snapshot
	s.hasLatest = true
	s.lastError = ""
}

type SamplerStatus struct {
	Latest      Snapshot  `json:"latest"`
	HasLatest   bool      `json:"hasLatest"`
	LastError   string    `json:"lastError,omitempty"`
	LastFailure time.Time `json:"lastFailure,omitempty"`
}

func (s *Sampler) Status() SamplerStatus {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return SamplerStatus{
		Latest:      s.latest,
		HasLatest:   s.hasLatest,
		LastError:   s.lastError,
		LastFailure: s.lastFailure,
	}
}
