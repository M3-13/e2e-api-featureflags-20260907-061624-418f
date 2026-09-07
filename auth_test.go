package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestAuthMissingToken(t *testing.T) {
	store := NewFlagStore()
	store.Create(Flag{Key: "feature", Enabled: true})
	mux := newMux(store, "secret", 0)

	req := httptest.NewRequest(http.MethodGet, "/flags", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
	if got := strings.TrimSpace(rec.Body.String()); got != `{"error":"unauthorized"}` {
		t.Fatalf("unexpected body: %q", got)
	}
}

func TestAuthWrongToken(t *testing.T) {
	store := NewFlagStore()
	mux := newMux(store, "secret", 0)

	req := httptest.NewRequest(http.MethodGet, "/flags", nil)
	req.Header.Set("Authorization", "Bearer wrong")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
}

func TestAuthCorrectToken(t *testing.T) {
	store := NewFlagStore()
	store.Create(Flag{Key: "feature", Enabled: true})
	mux := newMux(store, "secret", 0)

	req := httptest.NewRequest(http.MethodGet, "/flags", nil)
	req.Header.Set("Authorization", "Bearer secret")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}

func TestAuthHealthzPublic(t *testing.T) {
	mux := newMux(NewFlagStore(), "secret", 0)

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}

func TestAuthEvaluateProtected(t *testing.T) {
	store := NewFlagStore()
	store.Create(Flag{Key: "feature", Enabled: true, RolloutPercent: 100})
	mux := newMux(store, "secret", 0)

	req := httptest.NewRequest(http.MethodGet, "/flags/feature/evaluate?user=alice", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
}

func TestAuthNoKeySetAllowsAccess(t *testing.T) {
	store := NewFlagStore()
	store.Create(Flag{Key: "feature", Enabled: true})
	mux := newMux(store, "", 0)

	req := httptest.NewRequest(http.MethodGet, "/flags", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 without api key, got %d", rec.Code)
	}
}
