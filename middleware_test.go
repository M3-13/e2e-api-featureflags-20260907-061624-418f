package main

import (
	"bytes"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestWithLoggingCapturesMethodPathAndStatus(t *testing.T) {
	var buf bytes.Buffer
	log.SetOutput(&buf)
	defer log.SetOutput(nil)

	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
	})
	handler := withLogging(inner)

	req := httptest.NewRequest(http.MethodPost, "/flags?user=secret", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	line := buf.String()
	if !strings.Contains(line, http.MethodPost) {
		t.Fatalf("log line missing method %q: %q", http.MethodPost, line)
	}
	if !strings.Contains(line, "/flags") {
		t.Fatalf("log line missing path /flags: %q", line)
	}
	if !strings.Contains(line, "201") {
		t.Fatalf("log line missing status 201: %q", line)
	}
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d", rec.Code)
	}
}

func TestWithLoggingExcludesQueryParameters(t *testing.T) {
	var buf bytes.Buffer
	log.SetOutput(&buf)
	defer log.SetOutput(nil)

	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	handler := withLogging(inner)

	req := httptest.NewRequest(http.MethodGet, "/flags/my-key/evaluate?user=alice&role=admin", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	line := buf.String()
	if strings.Contains(line, "alice") {
		t.Fatalf("log line leaked user query value: %q", line)
	}
	if strings.Contains(line, "admin") {
		t.Fatalf("log line leaked role query value: %q", line)
	}
	if strings.Contains(line, "?") {
		t.Fatalf("log line contains query string: %q", line)
	}
	if !strings.Contains(line, "/flags/my-key/evaluate") {
		t.Fatalf("log line missing path: %q", line)
	}
}

func TestWithLoggingDefaultStatus(t *testing.T) {
	var buf bytes.Buffer
	log.SetOutput(&buf)
	defer log.SetOutput(nil)

	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("ok"))
	})
	handler := withLogging(inner)

	req := httptest.NewRequest(http.MethodGet, "/flags", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	line := buf.String()
	if !strings.Contains(line, "200") {
		t.Fatalf("log line missing default status 200: %q", line)
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
}
