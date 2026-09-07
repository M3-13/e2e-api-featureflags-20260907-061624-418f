package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRateLimitExceeded(t *testing.T) {
	store := NewFlagStore()
	store.Create(Flag{Key: "feature", Enabled: true})
	mux := newMux(store, "", 1)

	for i := 0; i < 2; i++ {
		req := httptest.NewRequest(http.MethodGet, "/flags", nil)
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)

		if i == 0 {
			if rec.Code != http.StatusOK {
				t.Fatalf("first request: expected 200, got %d", rec.Code)
			}
			continue
		}
		if rec.Code != http.StatusTooManyRequests {
			t.Fatalf("request %d: expected 429, got %d", i+1, rec.Code)
		}
		if got := strings.TrimSpace(rec.Body.String()); got != `{"error":"too many requests"}` {
			t.Fatalf("unexpected body: %q", got)
		}
	}
}

func TestRateLimitHealthzUnaffected(t *testing.T) {
	mux := newMux(NewFlagStore(), "", 1)

	for i := 0; i < 5; i++ {
		req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("healthz request %d: expected 200, got %d", i+1, rec.Code)
		}
	}
}

func TestRateLimitDisabled(t *testing.T) {
	store := NewFlagStore()
	store.Create(Flag{Key: "feature", Enabled: true})
	mux := newMux(store, "", 0)

	for i := 0; i < 10; i++ {
		req := httptest.NewRequest(http.MethodGet, "/flags", nil)
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("request %d: expected 200, got %d", i+1, rec.Code)
		}
	}
}
