package store

import (
	"errors"
	"sync"
)

// ErrDuplicate is returned by Create when a flag with the same key already
// exists.
var ErrDuplicate = errors.New("flag already exists")

// Flag is a single feature flag held in memory.
type Flag struct {
	Key            string  `json:"key"`
	Enabled        bool    `json:"enabled"`
	Description    string  `json:"description"`
	RolloutPercent float64 `json:"rollout_percent"`
}

// Store is a thread-safe in-memory collection of feature flags.
type Store struct {
	mu    sync.RWMutex
	flags map[string]Flag
}

// NewStore returns an empty, ready-to-use Store.
func NewStore() *Store {
	return &Store{flags: make(map[string]Flag)}
}

// Create adds a new flag. It returns ErrDuplicate if a flag with the same key
// already exists.
func (s *Store) Create(f Flag) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.flags[f.Key]; exists {
		return ErrDuplicate
	}
	s.flags[f.Key] = f
	return nil
}

// List returns all flags. The order is not guaranteed.
func (s *Store) List() []Flag {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]Flag, 0, len(s.flags))
	for _, f := range s.flags {
		out = append(out, f)
	}
	return out
}

// Get returns the flag stored under key and whether it exists.
func (s *Store) Get(key string) (Flag, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	f, ok := s.flags[key]
	return f, ok
}

// Update replaces the flag stored under key. It returns the stored flag and
// whether the key existed.
func (s *Store) Update(key string, f Flag) (Flag, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.flags[key]; !exists {
		return Flag{}, false
	}
	f.Key = key
	s.flags[key] = f
	return f, true
}

// Delete removes the flag stored under key. It reports whether the key
// existed.
func (s *Store) Delete(key string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.flags[key]; !exists {
		return false
	}
	delete(s.flags, key)
	return true
}
