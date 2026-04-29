package main

import (
	"sync"
	"time"
)

type RequestStats struct {
	mu          sync.Mutex
	Total       int64
	Today       int64
	Day         string
	StartedAt   time.Time
	LastRequest time.Time
}

type RequestStatsSnapshot struct {
	Total       int64  `json:"total"`
	Today       int64  `json:"today"`
	Day         string `json:"day"`
	StartedAt   string `json:"startedAt"`
	LastRequest string `json:"lastRequest,omitempty"`
}

func NewRequestStats() *RequestStats {
	now := time.Now()
	return &RequestStats{
		Day:       now.Format("2006-01-02"),
		StartedAt: now,
	}
}

func (s *RequestStats) Inc() {
	now := time.Now()
	day := now.Format("2006-01-02")

	s.mu.Lock()
	defer s.mu.Unlock()
	if s.Day != day {
		s.Day = day
		s.Today = 0
	}
	s.Total++
	s.Today++
	s.LastRequest = now
}

func (s *RequestStats) Snapshot() RequestStatsSnapshot {
	now := time.Now()
	day := now.Format("2006-01-02")

	s.mu.Lock()
	defer s.mu.Unlock()
	if s.Day != day {
		s.Day = day
		s.Today = 0
	}
	last := ""
	if !s.LastRequest.IsZero() {
		last = s.LastRequest.Format(time.RFC3339)
	}
	return RequestStatsSnapshot{
		Total:       s.Total,
		Today:       s.Today,
		Day:         s.Day,
		StartedAt:   s.StartedAt.Format(time.RFC3339),
		LastRequest: last,
	}
}
