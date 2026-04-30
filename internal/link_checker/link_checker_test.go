package link_checker

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync/atomic"
	"testing"

	"github.com/hhow09/page-insight-tool/internal/config"
	"github.com/hhow09/page-insight-tool/internal/httpclient"
)

func TestSummarize_internalExternal_and_inaccessible(t *testing.T) {
	t.Parallel()
	remote := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/ok":
			w.WriteHeader(http.StatusOK)
		case "/gone":
			http.NotFound(w, r)
		default:
			http.Error(w, "err", http.StatusInternalServerError)
		}
	}))
	t.Cleanup(remote.Close)

	origin, err := url.Parse(remote.URL + "/dir/page.html")
	if err != nil {
		t.Fatal(err)
	}

	cfg := config.Default()
	lc := &LinkChecker{
		client: httpclient.New(&cfg.HTTPClient),
		cfg:    &cfg.Link,
	}

	raw := []string{
		"../ok",          // internal
		"/gone",          // inaccessible
		"mailto:x@y.com", // skipped non-navigable
		"#frag",          // skipped non-navigable
		remote.URL + "/ok",
	}
	rep, err := lc.Summarize(context.Background(), origin, raw)
	if err != nil {
		t.Fatal(err)
	}
	if rep.SkippedNonNavigable != 2 {
		t.Fatalf("SkippedNonNavigable: got %d want 2", rep.SkippedNonNavigable)
	}
	if rep.InternalLinks != 3 {
		t.Fatalf("InternalLinks: got %d want 3 (../ok, /gone, abs same host)", rep.InternalLinks)
	}
	if rep.ExternalLinks != 0 {
		t.Fatalf("ExternalLinks: got %d", rep.ExternalLinks)
	}
	if rep.InaccessibleLinks != 1 {
		t.Fatalf("InaccessibleLinks: got %d want 1 (/gone -> 404)", rep.InaccessibleLinks)
	}
}

func TestSummarize_nil_page(t *testing.T) {
	t.Parallel()
	cfg := config.Default()
	lc := New(httpclient.New(&cfg.HTTPClient), &cfg.Link)
	summary, err := lc.Summarize(context.Background(), nil, []string{"/"})
	if err == nil {
		t.Fatal("expected error")
	}
	if summary != nil {
		t.Fatalf("expected nil summary, got %+v", summary)
	}
}

func TestSummarize_deduplication(t *testing.T) {
	t.Parallel()
	var probeCount atomic.Int32
	remote := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/gone" {
			probeCount.Add(1)
			http.NotFound(w, r)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(remote.Close)

	origin, _ := url.Parse(remote.URL + "/")
	cfg := config.Default()

	raw := []string{
		"/gone",
		"/gone#frag",
		"/gone",
		"/ok",
	}

	lc := New(httpclient.New(&cfg.HTTPClient), &cfg.Link)
	summary, err := lc.Summarize(context.Background(), origin, raw)
	if err != nil {
		t.Fatal(err)
	}
	// We expect 3 InaccessibleLinks because /gone appears 3 times.
	if summary.InaccessibleLinks != 3 {
		t.Fatalf("InaccessibleLinks: got %d want 3", summary.InaccessibleLinks)
	}
	// network probe should happen only once for /gone.
	// (#fragment is de-duplicated in network probe)
	if probeCount.Load() != 1 {
		t.Fatalf("probeCount: got %d want 1 (should be cached), got %d", probeCount.Load(), probeCount.Load())
	}
}
