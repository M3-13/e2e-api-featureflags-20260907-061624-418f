package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
)

func newTestMux() *http.ServeMux {
	return newMux(NewFlagStore())
}

func seedFlag(t *testing.T, s *FlagStore, key string, rolloutPercent int) {
	t.Helper()
	if err := s.Create(Flag{Key: key, Enabled: true, RolloutPercent: rolloutPercent}); err != nil {
		t.Fatalf("seed: %v", err)
	}
}

func TestEvaluateDeterministic(t *testing.T) {
	s := NewFlagStore()
	seedFlag(t, s, "feature", 50)
	mux := newMux(s)

	var results []bool
	for i := 0; i < 10; i++ {
		req := httptest.NewRequest(http.MethodGet, "/flags/feature/evaluate?user=alice", nil)
		rr := httptest.NewRecorder()
		mux.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200", rr.Code)
		}
		var body map[string]bool
		if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		results = append(results, body["enabled"])
	}
	for _, r := range results[1:] {
		if r != results[0] {
			t.Fatalf("inconsistent results: %v", results)
		}
	}
}

func TestEvaluateDistributionApproximatesRolloutPercent(t *testing.T) {
	const rolloutPercent = 30
	const numUsers = 1000
	s := NewFlagStore()
	seedFlag(t, s, "feature", rolloutPercent)
	mux := newMux(s)

	enabledCount := 0
	for i := 0; i < numUsers; i++ {
		req := httptest.NewRequest(http.MethodGet, "/flags/feature/evaluate?user=user"+strconv.Itoa(i), nil)
		rr := httptest.NewRecorder()
		mux.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200", rr.Code)
		}
		var body map[string]bool
		if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if body["enabled"] {
			enabledCount++
		}
	}

	gotPercent := float64(enabledCount) / numUsers * 100
	wantPercent := float64(rolloutPercent)
	if diff := gotPercent - wantPercent; diff > 5 || diff < -5 {
		t.Fatalf("distribution %.2f%% deviates too far from %d%%", gotPercent, rolloutPercent)
	}
}

func TestEvaluateMissingUser(t *testing.T) {
	s := NewFlagStore()
	seedFlag(t, s, "feature", 50)
	mux := newMux(s)

	req := httptest.NewRequest(http.MethodGet, "/flags/feature/evaluate", nil)
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rr.Code)
	}

	req = httptest.NewRequest(http.MethodGet, "/flags/feature/evaluate?user=", nil)
	rr = httptest.NewRecorder()
	mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("empty user: status = %d, want 400", rr.Code)
	}
}

func TestEvaluateUnknownKey(t *testing.T) {
	mux := newTestMux()

	req := httptest.NewRequest(http.MethodGet, "/flags/nope/evaluate?user=alice", nil)
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rr.Code)
	}
}
