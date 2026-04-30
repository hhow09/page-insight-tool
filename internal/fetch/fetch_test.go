package fetch

import (
	"bytes"
	"compress/gzip"
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/hhow09/page-insight-tool/internal/config"
	"github.com/hhow09/page-insight-tool/internal/httpclient"
	"github.com/hhow09/page-insight-tool/internal/model"
)

func testFetch(ctx context.Context, raw string, cfg *config.Config) (*model.FetchResult, error) {
	client := httpclient.New(&cfg.HTTPClient)
	fetcher := &Fetcher{
		client: client,
		cfg:    &cfg.Fetch,
	}
	return fetcher.Fetch(ctx, raw)
}

func TestFetch_OK(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", htmlContentType)
		_, _ = w.Write([]byte("<html><title>Hi</title></html>"))
	}))
	t.Cleanup(srv.Close)

	cfg := config.Default()

	res, err := testFetch(context.Background(), srv.URL, cfg)
	if err != nil {
		t.Fatal(err)
	}
	if res.Status != http.StatusOK {
		t.Fatalf("status: got %d want %d", res.Status, http.StatusOK)
	}
	if !strings.Contains(string(res.Body), "<title>Hi</title>") {
		t.Fatalf("body: %q", res.Body)
	}
	if want, got := srv.URL, res.FinalURL.String(); want != got {
		t.Fatalf("FinalURL: got %q want %q", got, want)
	}
}

func TestFetch_redirectFinalURL(t *testing.T) {
	t.Parallel()
	final := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("<html>final</html>"))
	}))
	mid := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, final.URL, http.StatusFound)
	}))
	first := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, mid.URL, http.StatusFound)
	}))
	t.Cleanup(first.Close)
	t.Cleanup(mid.Close)
	t.Cleanup(final.Close)

	cfg := config.Default()
	cfg.HTTPClient.Timeout = 5 * time.Second

	res, err := testFetch(context.Background(), first.URL, cfg)
	if err != nil {
		t.Fatal(err)
	}
	if got := res.FinalURL.String(); got != final.URL {
		t.Fatalf("FinalURL: got %q want %q", got, final.URL)
	}
}

func TestFetch_tooManyRedirects(t *testing.T) {
	t.Parallel()
	var a, b *httptest.Server
	a = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, b.URL, http.StatusFound)
	}))
	t.Cleanup(a.Close)

	b = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, a.URL, http.StatusFound)
	}))
	t.Cleanup(b.Close)

	cfg := config.Default()
	cfg.HTTPClient.MaxRedirects = 4

	_, err := testFetch(context.Background(), a.URL, cfg)
	if err == nil {
		t.Fatal("expected error")
	}
	var fe *FetchError
	if !errors.As(err, &fe) || !strings.Contains(strings.ToLower(fe.Message), "redirect") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestFetch_404(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", htmlContentType)
		w.WriteHeader(http.StatusNotFound)
	}))
	t.Cleanup(srv.Close)

	cfg := config.Default()
	_, err := testFetch(context.Background(), srv.URL, cfg)
	var fe *FetchError
	if !errors.As(err, &fe) {
		t.Fatalf("want *FetchError, got %T %v", err, err)
	}
	if fe.HTTPStatus != http.StatusNotFound {
		t.Fatalf("HTTPStatus: got %d want %d", fe.HTTPStatus, http.StatusNotFound)
	}
}

func TestFetch_invalidScheme(t *testing.T) {
	t.Parallel()
	cfg := config.Default()
	_, err := testFetch(context.Background(), "ftp://example.com/", cfg)
	var fe *FetchError
	if !errors.As(err, &fe) {
		t.Fatalf("got %T %v", err, err)
	}
	if !strings.Contains(fe.Message, "only http and https URLs are supported") {
		t.Fatalf("message: %q", fe.Message)
	}
}

func TestFetch_contextDeadline(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		time.Sleep(2 * time.Second)
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(srv.Close)

	cfg := config.Default()
	cfg.HTTPClient.Timeout = time.Hour

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	_, err := testFetch(ctx, srv.URL, cfg)
	if err == nil {
		t.Fatal("expected timeout")
	}
	var fe *FetchError
	if !errors.As(err, &fe) {
		t.Fatalf("got %T %v", err, err)
	}
	low := strings.ToLower(fe.Message)
	if !strings.Contains(low, "deadline") && !strings.Contains(low, "canceled") {
		t.Fatalf("message: %q", fe.Message)
	}
}

func TestFetch_maxBodyBytes(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		for i := 0; i < 2000; i++ {
			_, _ = fmt.Fprint(w, "x")
		}
	}))
	t.Cleanup(srv.Close)

	cfg := config.Default()
	cfg.Fetch.MaxBodyBytes = 100

	_, err := testFetch(context.Background(), srv.URL, cfg)
	if err == nil {
		t.Fatal("expected error from MaxBytesReader")
	}
}

// TestFetch_edgeCases covers odd but common HTTP behaviors using httptest
func TestFetch_edgeCases(t *testing.T) {
	t.Parallel()
	cfg := config.Default()
	t.Run("non_html_content_type", func(t *testing.T) {
		t.Parallel()
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{}`))
		}))
		t.Cleanup(srv.Close)

		_, err := testFetch(context.Background(), srv.URL, cfg)
		var fe *FetchError
		if !errors.As(err, &fe) || fe.Message != "Only HTML is supported" {
			t.Fatalf("got %v, want Only HTML is supported error", err)
		}
	})

	t.Run("empty_URL", func(t *testing.T) {
		t.Parallel()
		_, err := testFetch(context.Background(), "", cfg)
		var fe *FetchError
		if !errors.As(err, &fe) {
			t.Fatalf("got %T %v", err, err)
		}
	})

	t.Run("missing_host", func(t *testing.T) {
		t.Parallel()
		_, err := testFetch(context.Background(), "http:///nohost", cfg)
		var fe *FetchError
		if !errors.As(err, &fe) || !strings.Contains(fe.Message, "missing a host") {
			t.Fatalf("got %v", err)
		}
	})

	t.Run("204_no_content", func(t *testing.T) {
		t.Parallel()
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", htmlContentType)
			w.WriteHeader(http.StatusNoContent)
		}))
		t.Cleanup(srv.Close)

		res, err := testFetch(context.Background(), srv.URL, cfg)
		if err != nil {
			t.Fatal(err)
		}
		if res.Status != http.StatusNoContent {
			t.Fatalf("status: got %d", res.Status)
		}
		if len(res.Body) != 0 {
			t.Fatalf("body: want empty, got %d bytes", len(res.Body))
		}
	})

	t.Run("200_empty_body", func(t *testing.T) {
		t.Parallel()
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", htmlContentType)
			w.WriteHeader(http.StatusOK)
		}))
		t.Cleanup(srv.Close)
		cfg := config.Default()
		res, err := testFetch(context.Background(), srv.URL, cfg)
		if err != nil {
			t.Fatal(err)
		}
		if len(res.Body) != 0 {
			t.Fatalf("body: want empty, got %d bytes", len(res.Body))
		}
	})

	t.Run("gzip_content_encoding", func(t *testing.T) {
		t.Parallel()
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.Header().Set("Content-Encoding", "gzip")
			var buf bytes.Buffer
			gw := gzip.NewWriter(&buf)
			_, _ = gw.Write([]byte("<html><p>gzip-payload</p></html>"))
			_ = gw.Close()
			_, _ = w.Write(buf.Bytes())
		}))
		t.Cleanup(srv.Close)
		cfg := config.Default()
		res, err := testFetch(context.Background(), srv.URL, cfg)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(res.Body), "gzip-payload") {
			t.Fatalf("body: %q", res.Body)
		}
	})

	t.Run("relative_redirect", func(t *testing.T) {
		t.Parallel()
		mux := http.NewServeMux()
		mux.HandleFunc("/start", func(w http.ResponseWriter, r *http.Request) {
			http.Redirect(w, r, "/end", http.StatusFound)
		})
		mux.HandleFunc("/end", func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", htmlContentType)
			_, _ = w.Write([]byte("<html>after-relative</html>"))
		})
		srv := httptest.NewServer(mux)
		t.Cleanup(srv.Close)
		cfg := config.Default()
		res, err := testFetch(context.Background(), srv.URL+"/start", cfg)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.HasSuffix(res.FinalURL.Path, "/end") {
			t.Fatalf("FinalURL.Path: %s", res.FinalURL.Path)
		}
		if !strings.Contains(string(res.Body), "after-relative") {
			t.Fatalf("body: %q", res.Body)
		}
	})

	t.Run("query_string_preserved", func(t *testing.T) {
		t.Parallel()
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", htmlContentType)
			if r.URL.Query().Get("q") != "edge" {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			_, _ = w.Write([]byte("ok"))
		}))
		t.Cleanup(srv.Close)
		cfg := config.Default()
		res, err := testFetch(context.Background(), srv.URL+"?q=edge&x=1", cfg)
		if err != nil {
			t.Fatal(err)
		}
		if string(res.Body) != "ok" {
			t.Fatalf("body: %q", res.Body)
		}
	})

	t.Run("fragment_not_sent_to_server", func(t *testing.T) {
		t.Parallel()
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", htmlContentType)
			if r.URL.Path != "/doc" {
				w.WriteHeader(http.StatusNotFound)
				return
			}
			if r.URL.Fragment != "" {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			_, _ = w.Write([]byte("doc"))
		}))
		t.Cleanup(srv.Close)
		cfg := config.Default()

		res, err := testFetch(context.Background(), srv.URL+"/doc#section", cfg)
		if err != nil {
			t.Fatal(err)
		}
		if string(res.Body) != "doc" {
			t.Fatalf("body: %q", res.Body)
		}
	})

	t.Run("201_created", func(t *testing.T) {
		t.Parallel()
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Location", "/new/1")
			w.Header().Set("Content-Type", htmlContentType)
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"id":1}`))
		}))
		t.Cleanup(srv.Close)
		cfg := config.Default()

		res, err := testFetch(context.Background(), srv.URL, cfg)
		if err != nil {
			t.Fatal(err)
		}
		if res.Status != http.StatusCreated {
			t.Fatalf("status: got %d want %d", res.Status, http.StatusCreated)
		}
	})

	t.Run("304_not_modified_fails_policy", func(t *testing.T) {
		t.Parallel()
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusNotModified)
		}))
		t.Cleanup(srv.Close)
		cfg := config.Default()

		_, err := testFetch(context.Background(), srv.URL, cfg)
		var fe *FetchError
		if !errors.As(err, &fe) || fe.HTTPStatus != http.StatusNotModified {
			t.Fatalf("want 304 FetchError, got %v", err)
		}
	})

	t.Run("503_service_unavailable", func(t *testing.T) {
		t.Parallel()
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusServiceUnavailable)
		}))
		t.Cleanup(srv.Close)
		cfg := config.Default()
		_, err := testFetch(context.Background(), srv.URL, cfg)
		var fe *FetchError
		if !errors.As(err, &fe) || fe.HTTPStatus != http.StatusServiceUnavailable {
			t.Fatalf("want 503 FetchError, got %v", err)
		}
	})

	t.Run("redirects_at_max_minus_one_ok", func(t *testing.T) {
		t.Parallel()
		final := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", htmlContentType)
			_, _ = w.Write([]byte("end"))
		}))
		mid := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.Redirect(w, r, final.URL, http.StatusFound)
		}))
		first := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.Redirect(w, r, mid.URL, http.StatusMovedPermanently)
		}))
		t.Cleanup(first.Close)
		t.Cleanup(mid.Close)
		t.Cleanup(final.Close)

		cfg := config.Default()

		cfg.HTTPClient.MaxRedirects = 3
		// Two redirect responses then 200: need MaxRedirects at least 3 for this client policy.
		res, err := testFetch(context.Background(), first.URL, cfg)
		if err != nil {
			t.Fatal(err)
		}
		if string(res.Body) != "end" {
			t.Fatalf("body: %q", res.Body)
		}
	})
}

// TestFetch_integration_realWorld exercises Fetch against public HTTPS endpoints.
// Requires outbound network; skipped when tests run with -short.
func TestFetch_integration_realWorld(t *testing.T) {
	if testing.Short() {
		t.Skip("skip network integration tests (run without -short to enable)")
	}

	cfg := config.Default()
	cfg.HTTPClient.Timeout = 30 * time.Second

	cases := []struct {
		name string
		url  string
	}{
		{name: "iana_example_https", url: "https://example.com/"},
		{name: "iana_example_http", url: "http://example.com/"},
		{name: "w3c_www", url: "https://www.w3.org/"},
		{name: "ietf_rfc_editor", url: "https://www.rfc-editor.org/"},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ctx, cancel := context.WithTimeout(context.Background(), cfg.HTTPClient.Timeout)
			defer cancel()

			res, err := testFetch(ctx, tc.url, cfg)
			if err != nil {
				t.Fatalf("Fetch: %v", err)
			}
			if res.Status != http.StatusOK {
				t.Fatalf("Status: got %d want %d", res.Status, http.StatusOK)
			}
		})
	}
}

// TestFetch_integration_non2xx checks that real origins returning non-2xx surface *FetchError with HTTPStatus set.
// Requires outbound network; skipped when tests run with -short.
func TestFetch_integration_non2xx(t *testing.T) {
	if testing.Short() {
		t.Skip("skip network integration tests (run without -short to enable)")
	}

	cfg := config.Default()
	cfg.HTTPClient.Timeout = 30 * time.Second
	ctx, cancel := context.WithTimeout(context.Background(), cfg.HTTPClient.Timeout)
	defer cancel()

	// httpbin.org documents fixed status responses; useful for integration 404/418 checks.
	const url404 = "https://httpbin.org/status/404"
	_, err := testFetch(ctx, url404, cfg)
	var fe *FetchError
	if !errors.As(err, &fe) {
		t.Fatalf("want *FetchError, got %T %v", err, err)
	}
	if fe.HTTPStatus != http.StatusNotFound {
		t.Fatalf("HTTPStatus: got %d want %d", fe.HTTPStatus, http.StatusNotFound)
	}

	ctx2, cancel2 := context.WithTimeout(context.Background(), cfg.HTTPClient.Timeout)
	defer cancel2()
	const url418 = "https://httpbin.org/status/418"
	_, err418 := testFetch(ctx2, url418, cfg)
	var fe418 *FetchError
	if !errors.As(err418, &fe418) {
		t.Fatalf("418: want *FetchError, got %T %v", err418, err418)
	}
	if fe418.HTTPStatus != 418 {
		t.Fatalf("HTTPStatus: got %d want 418", fe418.HTTPStatus)
	}
}
