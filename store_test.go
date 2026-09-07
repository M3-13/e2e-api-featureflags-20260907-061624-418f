package main

import (
	"fmt"
	"sync"
	"testing"
)

func TestCreateAndGet(t *testing.T) {
	s := NewFlagStore()
	f := Flag{Key: "a", Enabled: true, Description: "desc", RolloutPercent: 50}
	if err := s.Create(f); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got, ok := s.Get("a")
	if !ok {
		t.Fatalf("expected flag to exist")
	}
	if got != f {
		t.Fatalf("got %+v, want %+v", got, f)
	}
}

func TestCreateConflict(t *testing.T) {
	s := NewFlagStore()
	if err := s.Create(Flag{Key: "a"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := s.Create(Flag{Key: "a"}); err != ErrFlagConflict {
		t.Fatalf("expected ErrFlagConflict, got %v", err)
	}
}

func TestListSorted(t *testing.T) {
	s := NewFlagStore()
	for _, k := range []string{"b", "a", "c"} {
		if err := s.Create(Flag{Key: k}); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	}
	got := s.List()
	if len(got) != 3 {
		t.Fatalf("expected 3 flags, got %d", len(got))
	}
	if got[0].Key != "a" || got[1].Key != "b" || got[2].Key != "c" {
		t.Fatalf("not sorted by key: %v", got)
	}
}

func TestListEmpty(t *testing.T) {
	s := NewFlagStore()
	got := s.List()
	if got == nil || len(got) != 0 {
		t.Fatalf("expected empty non-nil list, got %#v", got)
	}
}

func TestUpdate(t *testing.T) {
	s := NewFlagStore()
	if err := s.Create(Flag{Key: "a", Enabled: false}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	f, ok := s.Update(Flag{Key: "a", Enabled: true, Description: "updated", RolloutPercent: 10})
	if !ok {
		t.Fatalf("expected update to succeed")
	}
	if !f.Enabled || f.Description != "updated" || f.RolloutPercent != 10 {
		t.Fatalf("unexpected updated flag: %+v", f)
	}
	if _, ok := s.Update(Flag{Key: "missing"}); ok {
		t.Fatalf("expected update of missing key to fail")
	}
}

func TestDelete(t *testing.T) {
	s := NewFlagStore()
	if err := s.Create(Flag{Key: "a"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !s.Delete("a") {
		t.Fatalf("expected delete to succeed")
	}
	if _, ok := s.Get("a"); ok {
		t.Fatalf("expected flag to be gone")
	}
	if s.Delete("a") {
		t.Fatalf("expected delete of missing key to fail")
	}
}

func TestConcurrentAccess(t *testing.T) {
	s := NewFlagStore()
	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			key := fmt.Sprintf("key-%d", i%10)
			_ = s.Create(Flag{Key: key, Enabled: true})
			_, _ = s.Get(key)
			_ = s.List()
		}(i)
	}
	wg.Wait()

	got := s.List()
	if len(got) != 10 {
		t.Fatalf("expected 10 flags after concurrent writes, got %d", len(got))
	}
}
