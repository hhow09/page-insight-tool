// Package server wires the HTTP listener and routes for page-insight-tool.
package server

import (
	"net/http"
	"time"

	"github.com/hhow09/page-insight-tool/internal/config"
)

// Run listens on addr and blocks until the server stops or returns an error.
// Phase 1 exposes only GET /health (200 OK, JSON body). Static UI is added later.
func Run(addr string, config *config.Config) error {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", healthHandler)
	mux.HandleFunc("/api/analyze", GetResolveHandler(config))

	srv := &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
	}
	return srv.ListenAndServe()
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"status":"ok"}` + "\n"))
}
