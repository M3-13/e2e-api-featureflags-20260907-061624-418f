package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHealthz(t *testing.T) {
	mux := newMux(NewFlagStore())
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
	if got := strings.TrimSpace(rec.Body.String()); got != `{"status":"ok"}` {
		t.Fatalf("unexpected body: %q", got)
	}
}

func TestHealthzNoCORSHeader(t *testing.T) {
	mux := newMux(NewFlagStore())
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Fatalf("unexpected CORS header: %q", got)
	}
}

func TestWriteError500Generic(t *testing.T) {
	rec := httptest.NewRecorder()
	writeError(rec, http.StatusInternalServerError, "secret internal detail: store.go:42 0xc0001a2b3c")

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected status 500, got %d", rec.Code)
	}
	body := rec.Body.String()
	if got := strings.TrimSpace(body); got != `{"error":"internal server error"}` {
		t.Fatalf("expected generic body, got %q", got)
	}
	if strings.Contains(body, "secret") || strings.Contains(body, "store.go") || strings.Contains(body, "0xc000") {
		t.Fatalf("leaked internal detail: %q", body)
	}
}

func TestWriteErrorClientErrorPreservesMessage(t *testing.T) {
	rec := httptest.NewRecorder()
	writeError(rec, http.StatusBadRequest, "invalid key")

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", rec.Code)
	}
	if got := strings.TrimSpace(rec.Body.String()); got != `{"error":"invalid key"}` {
		t.Fatalf("unexpected body: %q", got)
	}
}
