package window

import (
	"sync"
	"time"
)

type Window struct {
	ID             string    `json:"id"`
	Source         string    `json:"source"` // omar
	ExperimentType string    `json:"experiment_type"`
	TargetType     string    `json:"target_type"`
	TargetName     string    `json:"target_name"`
	TargetNS       string    `json:"target_namespace"`
	StartAt        time.Time `json:"start_at"`
	EndAt          time.Time `json:"end_at"`
	DurationSec    int       `json:"duration_sec"`
}

type Store struct {
	mu      sync.RWMutex
	windows []Window
}

func NewStore() *Store {
	return &Store{
		windows: make([]Window, 0, 128),
	}
}

func (s *Store) Add(w Window) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.windows = append(s.windows, w)
}

func (s *Store) List() []Window {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]Window, len(s.windows))
	copy(out, s.windows)
	return out
}

func (s *Store) Clear() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.windows = s.windows[:0]
}
