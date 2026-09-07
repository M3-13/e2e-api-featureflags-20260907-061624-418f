package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"
)

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Pragma", "no-cache")
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

func newMux(store *FlagStore, apiKey string, rps int) *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", handleHealthz)

	limiter := newTokenBucket(rps)

	protect := func(h http.Handler) http.Handler {
		h = withTokenBucket(h, limiter)
		if apiKey != "" {
			h = withAuth(h, apiKey)
		}
		return withLogging(h)
	}

	mux.Handle("POST /flags", protect(handleCreateFlag(store)))
	mux.Handle("GET /flags", protect(handleListFlags(store)))
	mux.Handle("GET /flags/{key}", protect(handleGetFlag(store)))
	mux.Handle("PUT /flags/{key}", protect(handleUpdateFlag(store)))
	mux.Handle("DELETE /flags/{key}", protect(handleDeleteFlag(store)))
	mux.Handle("GET /flags/{key}/evaluate", protect(handleEvaluate(store)))
	return mux
}

func main() {
	apiKey := os.Getenv("FLAG_API_KEY")
	rps := 100
	if v := os.Getenv("RATE_LIMIT_PER_SECOND"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			rps = n
		}
	}

	if apiKey == "" {
		log.Println("FLAG_API_KEY not set; refusing non-loopback binds")
	}

	const addr = "127.0.0.1:8080"

	mux := newMux(NewFlagStore(), apiKey, rps)

	srv := &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       60 * time.Second,
		MaxHeaderBytes:    1 << 20,
	}

	log.Printf("listening on %s", addr)

	cert := os.Getenv("TLS_CERT_FILE")
	key := os.Getenv("TLS_KEY_FILE")
	var err error
	if cert != "" && key != "" {
		err = srv.ListenAndServeTLS(cert, key)
	} else {
		err = srv.ListenAndServe()
	}
	if err != nil {
		log.Fatalf("server error: %v", err)
	}
}
