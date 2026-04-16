package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/hhow09/page-insight-tool/internal/config"
)

func TestResolve_MethodNotAllowed(t *testing.T) {
	t.Parallel()
	cfg := config.Default()
	handler := GetResolveHandler(cfg)

	req := httptest.NewRequest(http.MethodGet, "/api/analyze", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405 Method Not Allowed, got %d", rec.Code)
	}
}

func TestResolve_InvalidBody(t *testing.T) {
	t.Parallel()
	cfg := config.Default()
	handler := GetResolveHandler(cfg)

	req := httptest.NewRequest(http.MethodPost, "/api/analyze", strings.NewReader(`{invalid json`))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 Bad Request, got %d", rec.Code)
	}
}

func TestResolve_EmptyURL(t *testing.T) {
	t.Parallel()
	cfg := config.Default()
	handler := GetResolveHandler(cfg)

	req := httptest.NewRequest(http.MethodPost, "/api/analyze", strings.NewReader(`{"url": ""}`))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 Bad Request, got %d", rec.Code)
	}
}

func TestResolve_Success(t *testing.T) {
	t.Parallel()

	// External server mock
	extSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(extSrv.Close)

	// Create a different hostname out of the external server bound locally
	// httptest binds to 127.0.0.1:port, changing it to localhost:port changes the Host component.
	extURL := strings.Replace(extSrv.URL, "127.0.0.1", "localhost", 1)

	// Target server mock
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/dead-link" {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = fmt.Fprintf(w, `
			<!DOCTYPE html>
			<html>
			<head><title>Mock Page</title></head>
			<body>
				<h1>Welcome</h1>
				<a href="/">Internal Link</a>
				<a href="%s">External Link</a>
				<a href="/dead-link">Inaccessible Internal Link</a>
			</body>
			</html>
		`, extURL)
	}))
	t.Cleanup(srv.Close)

	cfg := config.Default()
	cfg.Fetch.Timeout = 1 * time.Second
	cfg.Link.PerLinkTimeout = 1 * time.Second
	cfg.Link.Workers = 2

	handler := GetResolveHandler(cfg)

	reqBody := fmt.Sprintf(`{"url": "%s"}`, srv.URL)
	req := httptest.NewRequest(http.MethodPost, "/api/analyze", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d. Body: %s", rec.Code, rec.Body.String())
	}

	var resp AnalyzeResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.Error != nil {
		t.Fatalf("expected nil error, got %s", *resp.Error)
	}
	if resp.Data == nil {
		t.Fatal("expected non-nil Data")
	}

	d := resp.Data
	if d.Title != "Mock Page" {
		t.Errorf("expected Title Mock Page, got %s", d.Title)
	}
	if d.HTMLVersion != "HTML5" {
		t.Errorf("expected HTML5, got %s", d.HTMLVersion)
	}
	if d.Headings.H1 != 1 {
		t.Errorf("expected 1 h1, got %d", d.Headings.H1)
	}

	// internal links: "/" and "/dead-link"
	if d.InternalLinks != 2 {
		t.Errorf("expected 2 InternalLinks, got %d", d.InternalLinks)
	}
	// external links: "http://localhost:<port>"
	if d.ExternalLinks != 1 {
		t.Errorf("expected 1 ExternalLinks, got %d", d.ExternalLinks)
	}
	// inaccessible links: only "/dead-link" should fail since others return 200
	if d.InaccessibleLinks != 1 {
		t.Errorf("expected 1 InaccessibleLinks, got %d", d.InaccessibleLinks)
	}
}

func TestResolve_FetchFails(t *testing.T) {
	t.Parallel()
	cfg := config.Default()
	handler := GetResolveHandler(cfg)

	// Fetch to a bad port that refuses connection, or a dummy URL that fails quickly
	reqBody := `{"url": "http://127.0.0.1:0"}`
	req := httptest.NewRequest(http.MethodPost, "/api/analyze", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	// In GetResolveHandler, fetch failures result in 404 StatusNotFound.
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404 StatusNotFound on fetch failure, got %d", rec.Code)
	}

	var resp AnalyzeResponse
	_ = json.NewDecoder(rec.Body).Decode(&resp)
	if resp.Error == nil {
		t.Fatal("expected an error string in response")
	}
}
