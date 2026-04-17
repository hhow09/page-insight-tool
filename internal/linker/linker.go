package linker

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/hhow09/page-insight-tool/internal/config"
)

type Summary struct {
	InternalLinks       int
	ExternalLinks       int
	SkippedNonNavigable int
	InaccessibleLinks   int
	PartialCheck        bool
}

// Summarize resolves raw hrefs against page, counts internal vs external, then probes unique http(s) URLs.
func Summarize(ctx context.Context, client *http.Client, page *url.URL, rawHrefs []string, lc *config.LinkConfig) (*Summary, error) {
	out, navigableURLs, err := collectClassifiedUnique(page, rawHrefs, lc.MaxURLsToCheck)
	if err != nil {
		return nil, err
	}

	inAccessibleLinks, probePartial, err := runProbes(ctx, client, navigableURLs, lc)
	if probePartial {
		out.PartialCheck = true
	}
	out.InaccessibleLinks = inAccessibleLinks
	return out, err
}

// dedupeKey returns the string representation of the URL without the fragment.
func dedupeKey(u *url.URL) string {
	if u == nil {
		return ""
	}
	nu := *u
	nu.Fragment = ""
	return nu.String()
}

// collectClassifiedUnique resolves each href, counts internal vs external and skips,
// and fills Summary with classification counts, LinksTotalUnique, LinksChecked, PartialCheck (when capped),
// and NavigableURLs (sorted deduped list to probe, after optional cap).
func collectClassifiedUnique(page *url.URL, rawHrefs []string, maxURLsToCheck int) (*Summary, []string, error) {
	if page == nil {
		return nil, nil, &url.Error{Op: "linker.collectClassifiedUnique", URL: "", Err: errors.New("nil page URL")}
	}
	slog.Debug("collectClassifiedUnique", "page", page.String(), "count", len(rawHrefs), "hrefs", rawHrefs)
	out := &Summary{}
	seen := make(map[string]struct{})
	var unique []string

	for _, href := range rawHrefs {
		href = strings.TrimSpace(href)
		if href == "" {
			continue
		}
		// Same-document fragment jump only; not fetched as a separate URL.
		if strings.HasPrefix(href, "#") {
			out.SkippedNonNavigable++
			continue
		}
		abs, e := Resolve(page, href)
		// Malformed reference or parse failure against the page URL.
		if e != nil {
			out.SkippedNonNavigable++
			continue
		}
		// mailto:, javascript:, tel:, etc. — excluded from probe list per plan.
		if !NavigableHTTP(abs) {
			out.SkippedNonNavigable++
			continue
		}
		// Same hostname as page (case-insensitive) → internal; otherwise external.
		if SameHost(page, abs) {
			out.InternalLinks++
		} else {
			out.ExternalLinks++
		}
		key := dedupeKey(abs)
		// Should not happen for http(s), but skip if we cannot form a stable dedupe string.
		if key == "" {
			continue
		}
		if _, dup := seen[key]; dup {
			continue
		}
		seen[key] = struct{}{}
		unique = append(unique, key)
	}

	sort.Strings(unique)
	if maxURLsToCheck > 0 && len(unique) > maxURLsToCheck {
		out.PartialCheck = true
		unique = unique[:maxURLsToCheck]
	}
	return out, unique, nil
}

// runProbeWorkerPool runs HEAD/GET accessibility checks for navigableURLs with a bounded worker pool.
func runProbes(ctx context.Context, client *http.Client, navigableURLs []string, lc *config.LinkConfig) (inaccessible int, partial bool, err error) {
	if len(navigableURLs) == 0 {
		return 0, false, nil
	}
	slog.Debug("navigableURLs collected", "count", len(navigableURLs), "urls", navigableURLs)

	jobs := make(chan string)
	var wg sync.WaitGroup
	var inaccessibleCount atomic.Int64

	for range lc.Workers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for linkURL := range jobs {
				if ctx.Err() != nil {
					return
				}
				inaccessible, _ := probeAccessibility(ctx, client, linkURL, lc)
				if inaccessible {
					inaccessibleCount.Add(1)
				}
			}
		}()
	}

send:
	for _, u := range navigableURLs {
		select {
		case <-ctx.Done(): // context canceled early exit
			partial = true
			break send
		case jobs <- u:
		}
	}
	close(jobs)
	wg.Wait()

	if ctx.Err() != nil { // context canceled
		partial = true
	}
	return int(inaccessibleCount.Load()), partial, ctx.Err()
}
