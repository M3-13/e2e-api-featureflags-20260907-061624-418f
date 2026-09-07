package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
)

func serve(t *testing.T, h http.HandlerFunc, method, target, body, pathKey string) *httptest.ResponseRecorder {
	t.Helper()
	var r io.Reader
	if body != "" {
		r = strings.NewReader(body)
	}
	req := httptest.NewRequest(method, target, r)
	if pathKey != "" {
		req.SetPathValue("key", pathKey)
	}
	rec := httptest.NewRecorder()
	h(rec, req)
	return rec
}

func decodeFlag(t *testing.T, rec *httptest.ResponseRecorder) Flag {
	t.Helper()
	var f Flag
	if err := json.Unmarshal(rec.Body.Bytes(), &f); err != nil {
		t.Fatalf("failed to decode flag response %q: %v", rec.Body.String(), err)
	}
	return f
}

func decodeError(t *testing.T, rec *httptest.ResponseRecorder) string {
	t.Helper()
	var e map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &e); err != nil {
		t.Fatalf("failed to decode error response %q: %v", rec.Body.String(), err)
	}
	return e["error"]
}

func TestCreateFlag(t *testing.T) {
	store := NewFlagStore()
	body := `{"key":"my_flag","enabled":true,"description":"a flag","rollout_percent":50}`
	rec := serve(t, handleCreateFlag(store), http.MethodPost, "/flags", body, "")

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
	f := decodeFlag(t, rec)
	if f.Key != "my_flag" || !f.Enabled || f.Description != "a flag" || f.RolloutPercent != 50 {
		t.Fatalf("unexpected flag: %+v", f)
	}
}

func TestCreateFlagRolloutPercentDefaultsToZero(t *testing.T) {
	store := NewFlagStore()
	rec := serve(t, handleCreateFlag(store), http.MethodPost, "/flags", `{"key":"k","enabled":true}`, "")
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", rec.Code)
	}
	f := decodeFlag(t, rec)
	if f.RolloutPercent != 0 {
		t.Fatalf("expected rollout_percent 0, got %d", f.RolloutPercent)
	}
}

func TestCreateFlagEmptyKey(t *testing.T) {
	store := NewFlagStore()
	rec := serve(t, handleCreateFlag(store), http.MethodPost, "/flags", `{"key":"","enabled":true}`, "")
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
	if decodeError(t, rec) == "" {
		t.Fatalf("expected non-empty error object")
	}
}

func TestCreateFlagInvalidKey(t *testing.T) {
	store := NewFlagStore()
	for _, key := range []string{"bad key", "with/slash", "dot.in", "über"} {
		body := `{"key":` + strconv.Quote(key) + `,"enabled":true}`
		rec := serve(t, handleCreateFlag(store), http.MethodPost, "/flags", body, "")
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("key %q: expected 400, got %d", key, rec.Code)
		}
	}
}

func TestCreateFlagMissingEnabled(t *testing.T) {
	store := NewFlagStore()
	rec := serve(t, handleCreateFlag(store), http.MethodPost, "/flags", `{"key":"k"}`, "")
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestCreateFlagRolloutPercentOutOfRange(t *testing.T) {
	store := NewFlagStore()
	for _, v := range []int{-1, 101} {
		body := `{"key":"k","enabled":true,"rollout_percent":` + strconv.Itoa(v) + `}`
		rec := serve(t, handleCreateFlag(store), http.MethodPost, "/flags", body, "")
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("rollout_percent %d: expected 400, got %d", v, rec.Code)
		}
	}
}

func TestCreateFlagConflict(t *testing.T) {
	store := NewFlagStore()
	body := `{"key":"dup","enabled":true}`
	if rec := serve(t, handleCreateFlag(store), http.MethodPost, "/flags", body, ""); rec.Code != http.StatusCreated {
		t.Fatalf("first create expected 201, got %d", rec.Code)
	}
	rec := serve(t, handleCreateFlag(store), http.MethodPost, "/flags", body, "")
	if rec.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d", rec.Code)
	}
}

func TestCreateFlagBodyTooLarge(t *testing.T) {
	store := NewFlagStore()
	body := `{"key":"k","enabled":true,"description":"` + strings.Repeat("a", maxBodyBytes+10) + `"}`
	rec := serve(t, handleCreateFlag(store), http.MethodPost, "/flags", body, "")
	if rec.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("expected 413, got %d", rec.Code)
	}
	if decodeError(t, rec) == "" {
		t.Fatalf("expected JSON error object")
	}
}

func TestCreateFlagDescriptionTooLong(t *testing.T) {
	store := NewFlagStore()
	body := `{"key":"k","enabled":true,"description":"` + strings.Repeat("a", 257) + `"}`
	rec := serve(t, handleCreateFlag(store), http.MethodPost, "/flags", body, "")
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
	if got := decodeError(t, rec); got != "description too long" {
		t.Fatalf("expected 'description too long', got %q", got)
	}
}

func TestUpdateFlagDescriptionTooLong(t *testing.T) {
	store := NewFlagStore()
	store.Create(Flag{Key: "k", Enabled: false})
	body := `{"enabled":true,"description":"` + strings.Repeat("a", 257) + `"}`
	rec := serve(t, handleUpdateFlag(store), http.MethodPut, "/flags/k", body, "k")
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
	if got := decodeError(t, rec); got != "description too long" {
		t.Fatalf("expected 'description too long', got %q", got)
	}
}

func TestCreateFlagDescriptionMaxLengthOK(t *testing.T) {
	store := NewFlagStore()
	body := `{"key":"k","enabled":true,"description":"` + strings.Repeat("a", 256) + `"}`
	rec := serve(t, handleCreateFlag(store), http.MethodPost, "/flags", body, "")
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", rec.Code)
	}
}

func TestCreateFlagLimitHandler(t *testing.T) {
	store := NewFlagStore()
	for i := 0; i < maxFlags; i++ {
		store.Create(Flag{Key: fmt.Sprintf("key-%d", i)})
	}
	rec := serve(t, handleCreateFlag(store), http.MethodPost, "/flags", `{"key":"overflow","enabled":true}`, "")
	if rec.Code != http.StatusInsufficientStorage {
		t.Fatalf("expected 507, got %d", rec.Code)
	}
	if got := decodeError(t, rec); got != "insufficient storage" {
		t.Fatalf("expected 'insufficient storage', got %q", got)
	}
}

func TestCreateFlagInvalidJSON(t *testing.T) {
	store := NewFlagStore()
	rec := serve(t, handleCreateFlag(store), http.MethodPost, "/flags", `not json`, "")
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestListFlags(t *testing.T) {
	store := NewFlagStore()
	store.Create(Flag{Key: "b", Enabled: true})
	store.Create(Flag{Key: "a", Enabled: false})

	rec := serve(t, handleListFlags(store), http.MethodGet, "/flags", "", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	var flags []Flag
	if err := json.Unmarshal(rec.Body.Bytes(), &flags); err != nil {
		t.Fatalf("failed to decode list: %v", err)
	}
	if len(flags) != 2 || flags[0].Key != "a" || flags[1].Key != "b" {
		t.Fatalf("unexpected list: %+v", flags)
	}
}

func TestListFlagsEmpty(t *testing.T) {
	store := NewFlagStore()
	rec := serve(t, handleListFlags(store), http.MethodGet, "/flags", "", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if got := strings.TrimSpace(rec.Body.String()); got != "[]" {
		t.Fatalf("expected empty array, got %q", got)
	}
}

func TestGetFlag(t *testing.T) {
	store := NewFlagStore()
	store.Create(Flag{Key: "k", Enabled: true, Description: "d"})
	rec := serve(t, handleGetFlag(store), http.MethodGet, "/flags/k", "", "k")
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	f := decodeFlag(t, rec)
	if f.Key != "k" || !f.Enabled || f.Description != "d" {
		t.Fatalf("unexpected flag: %+v", f)
	}
}

func TestGetFlagNotFound(t *testing.T) {
	store := NewFlagStore()
	rec := serve(t, handleGetFlag(store), http.MethodGet, "/flags/missing", "", "missing")
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
	if decodeError(t, rec) == "" {
		t.Fatalf("expected JSON error object")
	}
}

func TestUpdateFlag(t *testing.T) {
	store := NewFlagStore()
	store.Create(Flag{Key: "k", Enabled: false, Description: "old", RolloutPercent: 10})

	rec := serve(t, handleUpdateFlag(store), http.MethodPut, "/flags/k", `{"enabled":true,"description":"new","rollout_percent":80}`, "k")
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	f := decodeFlag(t, rec)
	if f.Key != "k" || !f.Enabled || f.Description != "new" || f.RolloutPercent != 80 {
		t.Fatalf("unexpected updated flag: %+v", f)
	}

	got, _ := store.Get("k")
	if !got.Enabled || got.Description != "new" || got.RolloutPercent != 80 {
		t.Fatalf("store not updated: %+v", got)
	}
}

func TestUpdateFlagNotFound(t *testing.T) {
	store := NewFlagStore()
	rec := serve(t, handleUpdateFlag(store), http.MethodPut, "/flags/missing", `{"enabled":true}`, "missing")
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
}

func TestUpdateFlagRolloutPercentOutOfRange(t *testing.T) {
	store := NewFlagStore()
	store.Create(Flag{Key: "k", Enabled: false})
	rec := serve(t, handleUpdateFlag(store), http.MethodPut, "/flags/k", `{"enabled":true,"rollout_percent":101}`, "k")
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestUpdateFlagMissingEnabled(t *testing.T) {
	store := NewFlagStore()
	store.Create(Flag{Key: "k", Enabled: false})
	rec := serve(t, handleUpdateFlag(store), http.MethodPut, "/flags/k", `{}`, "k")
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestUpdateFlagInvalidKey(t *testing.T) {
	store := NewFlagStore()
	rec := serve(t, handleUpdateFlag(store), http.MethodPut, "/flags/k", `{"enabled":true}`, "bad key")
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestDeleteFlag(t *testing.T) {
	store := NewFlagStore()
	store.Create(Flag{Key: "k", Enabled: true})
	rec := serve(t, handleDeleteFlag(store), http.MethodDelete, "/flags/k", "", "k")
	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", rec.Code)
	}
	if rec.Body.Len() != 0 {
		t.Fatalf("expected empty body, got %q", rec.Body.String())
	}
	if _, ok := store.Get("k"); ok {
		t.Fatalf("flag should be deleted")
	}
}

func TestDeleteFlagNotFound(t *testing.T) {
	store := NewFlagStore()
	rec := serve(t, handleDeleteFlag(store), http.MethodDelete, "/flags/missing", "", "missing")
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
}

func TestDeleteThenGetNotFound(t *testing.T) {
	store := NewFlagStore()
	store.Create(Flag{Key: "k", Enabled: true})
	serve(t, handleDeleteFlag(store), http.MethodDelete, "/flags/k", "", "k")
	rec := serve(t, handleGetFlag(store), http.MethodGet, "/flags/k", "", "k")
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404 after delete, got %d", rec.Code)
	}
}

func TestMethodNotAllowed(t *testing.T) {
	store := NewFlagStore()
	cases := []struct {
		h      http.HandlerFunc
		method string
		target string
		key    string
	}{
		{handleCreateFlag(store), http.MethodGet, "/flags", ""},
		{handleListFlags(store), http.MethodPost, "/flags", ""},
		{handleGetFlag(store), http.MethodPost, "/flags/k", "k"},
		{handleUpdateFlag(store), http.MethodPost, "/flags/k", "k"},
		{handleDeleteFlag(store), http.MethodGet, "/flags/k", "k"},
	}
	for _, c := range cases {
		rec := serve(t, c.h, c.method, c.target, "", c.key)
		if rec.Code != http.StatusMethodNotAllowed {
			t.Fatalf("%s %s: expected 405, got %d", c.method, c.target, rec.Code)
		}
		if decodeError(t, rec) == "" {
			t.Fatalf("%s %s: expected JSON error object", c.method, c.target)
		}
	}
}
