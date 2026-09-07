package main

import (
	"encoding/json"
	"log"
	"net/http"
)

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Printf("writeJSON: failed to encode response: %v", err)
	}
}

func writeError(w http.ResponseWriter, status int, msg string) {
	if status >= http.StatusInternalServerError {
		msg = "internal server error"
	}
	writeJSON(w, status, map[string]string{"error": msg})
}

func handleHealthz(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func newMux(store *FlagStore) *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", handleHealthz)
	mux.Handle("POST /flags", withLogging(handleCreateFlag(store)))
	mux.Handle("GET /flags", withLogging(handleListFlags(store)))
	mux.Handle("GET /flags/{key}", withLogging(handleGetFlag(store)))
	mux.Handle("PUT /flags/{key}", withLogging(handleUpdateFlag(store)))
	mux.Handle("DELETE /flags/{key}", withLogging(handleDeleteFlag(store)))
	mux.Handle("GET /flags/{key}/evaluate", withLogging(handleEvaluate(store)))
	return mux
}

func main() {
	mux := newMux(NewFlagStore())
	log.Println("listening on :8080")
	if err := http.ListenAndServe(":8080", mux); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
