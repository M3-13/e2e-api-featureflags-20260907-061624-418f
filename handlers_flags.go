package main

import (
	"encoding/json"
	"errors"
	"net/http"
	"regexp"
	"unicode/utf8"
)

const maxBodyBytes = 1 << 20

var keyPattern = regexp.MustCompile(`^[A-Za-z0-9_-]{1,64}$`)

func validKey(k string) bool {
	return keyPattern.MatchString(k)
}

type flagRequest struct {
	Key            string `json:"key"`
	Enabled        *bool  `json:"enabled"`
	Description    string `json:"description"`
	RolloutPercent *int   `json:"rollout_percent"`
}

func decodeBody(w http.ResponseWriter, r *http.Request, dst *flagRequest) bool {
	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
	if err := json.NewDecoder(r.Body).Decode(dst); err != nil {
		var maxBytesErr *http.MaxBytesError
		if errors.As(err, &maxBytesErr) {
			writeError(w, http.StatusRequestEntityTooLarge, "request body too large")
			return false
		}
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return false
	}
	return true
}

func validateFlagRequest(w http.ResponseWriter, req *flagRequest) bool {
	if req.Enabled == nil {
		writeError(w, http.StatusBadRequest, "enabled is required")
		return false
	}
	if req.RolloutPercent != nil && (*req.RolloutPercent < 0 || *req.RolloutPercent > 100) {
		writeError(w, http.StatusBadRequest, "rollout_percent must be between 0 and 100")
		return false
	}
	if utf8.RuneCountInString(req.Description) > 256 {
		writeError(w, http.StatusBadRequest, "description too long")
		return false
	}
	return true
}

func handleCreateFlag(s *FlagStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		var req flagRequest
		if !decodeBody(w, r, &req) {
			return
		}
		if !validKey(req.Key) {
			writeError(w, http.StatusBadRequest, "invalid key")
			return
		}
		if !validateFlagRequest(w, &req) {
			return
		}
		flag := Flag{
			Key:         req.Key,
			Enabled:     *req.Enabled,
			Description: req.Description,
		}
		if req.RolloutPercent != nil {
			flag.RolloutPercent = *req.RolloutPercent
		}
		if err := s.Create(flag); err != nil {
			if errors.Is(err, ErrFlagConflict) {
				writeError(w, http.StatusConflict, "flag already exists")
				return
			}
			if errors.Is(err, ErrFlagLimit) {
				writeJSON(w, http.StatusInsufficientStorage, map[string]string{"error": "insufficient storage"})
				return
			}
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusCreated, flag)
	}
}

func handleListFlags(s *FlagStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		writeJSON(w, http.StatusOK, s.List())
	}
}

func handleGetFlag(s *FlagStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		flag, ok := s.Get(r.PathValue("key"))
		if !ok {
			writeError(w, http.StatusNotFound, "flag not found")
			return
		}
		writeJSON(w, http.StatusOK, flag)
	}
}

func handleUpdateFlag(s *FlagStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		key := r.PathValue("key")
		if !validKey(key) {
			writeError(w, http.StatusBadRequest, "invalid key")
			return
		}
		var req flagRequest
		if !decodeBody(w, r, &req) {
			return
		}
		if !validateFlagRequest(w, &req) {
			return
		}
		flag := Flag{
			Key:         key,
			Enabled:     *req.Enabled,
			Description: req.Description,
		}
		if req.RolloutPercent != nil {
			flag.RolloutPercent = *req.RolloutPercent
		}
		updated, ok := s.Update(flag)
		if !ok {
			writeError(w, http.StatusNotFound, "flag not found")
			return
		}
		writeJSON(w, http.StatusOK, updated)
	}
}

func handleDeleteFlag(s *FlagStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		if !s.Delete(r.PathValue("key")) {
			writeError(w, http.StatusNotFound, "flag not found")
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}
