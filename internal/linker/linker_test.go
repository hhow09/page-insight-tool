package linker

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
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
	cfg.Link.MaxURLsToCheck = 0

	raw := []string{
		"../ok",          // internal
		"/gone",          // inaccessible
		"mailto:x@y.com", // skipped non-navigable
		"#frag",          // skipped non-navigable
		remote.URL + "/ok",
	}
	rep, err := Summarize(context.Background(), httpclient.New(&cfg.HTTPClient), origin, raw, &cfg.Link)
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

func TestSummarize_maxURLs_cap(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(srv.Close)
	page, _ := url.Parse(srv.URL + "/")
	var hrefs []string
	for i := range 10 {
		hrefs = append(hrefs, "/p"+strings.Repeat("a", i))
	}
	cfg := config.Default()
	cfg.Link.MaxURLsToCheck = 3
	rep, err := Summarize(context.Background(), httpclient.New(&cfg.HTTPClient), page, hrefs, &cfg.Link)
	if err != nil {
		t.Fatal(err)
	}
	if !rep.PartialCheck {
		t.Fatal("expected PartialCheck")
	}

}

func TestSummarize_nil_page(t *testing.T) {
	t.Parallel()
	cfg := config.Default()
	_, err := Summarize(context.Background(), httpclient.New(&cfg.HTTPClient), nil, []string{"/"}, &cfg.Link)
	if err == nil {
		t.Fatal("expected error")
	}
}
