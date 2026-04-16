package server

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHealth_GET(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(healthHandler))
	t.Cleanup(srv.Close)

	res, err := http.Get(srv.URL + "/health")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = res.Body.Close() })

	if res.StatusCode != http.StatusOK {
		t.Fatalf("status: got %d want %d", res.StatusCode, http.StatusOK)
	}
	body, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatal(err)
	}
	if ct := res.Header.Get("Content-Type"); ct != "application/json; charset=utf-8" {
		t.Fatalf("Content-Type: got %q want application/json", ct)
	}
	if string(body) != "{\"status\":\"ok\"}\n" {
		t.Fatalf("body: got %q", body)
	}
}

func TestHealth_nonGET(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(healthHandler))
	t.Cleanup(srv.Close)

	req, err := http.NewRequest(http.MethodPost, srv.URL+"/health", nil)
	if err != nil {
		t.Fatal(err)
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = res.Body.Close() })

	if res.StatusCode != http.StatusMethodNotAllowed {
		t.Fatalf("status: got %d want %d", res.StatusCode, http.StatusMethodNotAllowed)
	}
}
