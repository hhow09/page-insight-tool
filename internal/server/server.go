// Package server wires the HTTP listener and routes for page-insight-tool.
package server

import (
	"net/http"
	"time"

	"github.com/hhow09/page-insight-tool/internal/config"
)

type Server struct {
	cfg            *config.ServerConfig
	analyzeHandler http.Handler
}

func New(cfg *config.ServerConfig, analyzeHandler http.Handler) *Server {
	return &Server{
		cfg:            cfg,
		analyzeHandler: analyzeHandler,
	}
}

func (s *Server) Run() error {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", healthHandler)
	mux.Handle("/api/analyze", s.analyzeHandler)

	fs := http.FileServer(http.Dir("frontend/dist"))
	mux.Handle("/", fs)

	srv := &http.Server{
		Addr:              s.cfg.Addr,
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
