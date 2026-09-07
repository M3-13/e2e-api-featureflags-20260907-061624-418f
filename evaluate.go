package main

import (
	"hash/fnv"
	"net/http"
)

func handleEvaluate(s *FlagStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user := r.URL.Query().Get("user")
		if user == "" {
			writeError(w, http.StatusBadRequest, "user is required")
			return
		}
		if len(user) > 256 {
			writeError(w, http.StatusBadRequest, "user too long")
			return
		}

		key := r.PathValue("key")
		flag, ok := s.Get(key)
		if !ok {
			writeError(w, http.StatusNotFound, "flag not found")
			return
		}

		h := fnv.New32a()
		h.Write([]byte(key + "|" + user))
		bucket := int(h.Sum32() % 100)

		enabled := bucket < flag.RolloutPercent
		writeJSON(w, http.StatusOK, map[string]bool{"enabled": enabled})
	}
}
