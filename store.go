package main

import (
	"errors"
	"sort"
	"sync"
)

type Flag struct {
	Key            string `json:"key"`
	Enabled        bool   `json:"enabled"`
	Description    string `json:"description,omitempty"`
	RolloutPercent int    `json:"rollout_percent"`
}

type FlagStore struct {
	mu    sync.RWMutex
	flags map[string]Flag
}

var ErrFlagConflict = errors.New("flag already exists")

func NewFlagStore() *FlagStore {
	return &FlagStore{flags: make(map[string]Flag)}
}

func (s *FlagStore) Create(f Flag) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.flags[f.Key]; ok {
		return ErrFlagConflict
	}
	s.flags[f.Key] = f
	return nil
}

func (s *FlagStore) List() []Flag {
	s.mu.RLock()
	defer s.mu.RUnlock()
	keys := make([]string, 0, len(s.flags))
	for k := range s.flags {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	out := make([]Flag, 0, len(keys))
	for _, k := range keys {
		out = append(out, s.flags[k])
	}
	return out
}

func (s *FlagStore) Get(key string) (Flag, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	f, ok := s.flags[key]
	return f, ok
}

func (s *FlagStore) Update(f Flag) (Flag, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.flags[f.Key]; !ok {
		return Flag{}, false
	}
	s.flags[f.Key] = f
	return f, true
}

func (s *FlagStore) Delete(key string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.flags[key]; !ok {
		return false
	}
	delete(s.flags, key)
	return true
}
