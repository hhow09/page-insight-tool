package link_checker

import (
	"context"
	"errors"
	"log/slog"
	"maps"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"sync"

	"github.com/hhow09/page-insight-tool/internal/config"
)

type Summary struct {
	InternalLinks       int
	ExternalLinks       int
	SkippedNonNavigable int
	InaccessibleLinks   int
}

// Summarize resolves raw hrefs against page, counts internal vs external, then probes unique http(s) URLs.
func Summarize(ctx context.Context, client *http.Client, page *url.URL, rawHrefs []string, lc *config.LinkConfig) (*Summary, error) {
	summary, navigableURLs, err := collectNavigableLinks(page, rawHrefs)
	if err != nil {
		return nil, err
	}

	inAccessibleLinks, err := runProbes(ctx, client, navigableURLs, lc)
	summary.InaccessibleLinks = inAccessibleLinks
	return summary, err
}

// collectNavigableLinks resolves each href, counts internal vs external and skips,
// and fills Summary with classification counts,
// and NavigableURLs (all resolved URLs to probe).
func collectNavigableLinks(page *url.URL, rawHrefs []string) (*Summary, []*url.URL, error) {
	if page == nil {
		return nil, nil, &url.Error{Op: "link_checker.collectNavigableLinks", URL: "", Err: errors.New("nil page URL")}
	}
	slog.Debug("collectNavigableLinks", "page", page.String(), "count", len(rawHrefs), "hrefs", rawHrefs)
	out := &Summary{}
	var navigable []*url.URL

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
		abs, e := resolveRelative(page, href)
		// Malformed reference or parse failure against the page URL.
		if e != nil {
			out.SkippedNonNavigable++
			continue
		}
		// mailto:, javascript:, tel:, etc. — excluded from probe list per plan.
		if !navigableHTTP(abs) {
			out.SkippedNonNavigable++
			continue
		}
		// Same hostname as page (case-insensitive) → internal; otherwise external.
		if sameHost(page, abs) {
			out.InternalLinks++
		} else {
			out.ExternalLinks++
		}
		navigable = append(navigable, abs)
	}

	return out, navigable, nil
}

// runProbes runs HEAD/GET accessibility checks for navigableURLs with a bounded worker pool, returning the total count of inaccessible links.
func runProbes(ctx context.Context, client *http.Client, navigableURLs []*url.URL, lc *config.LinkConfig) (inaccessible int, err error) {
	if len(navigableURLs) == 0 {
		return 0, nil
	}
	toProbe := deduplicated(navigableURLs)
	slog.Debug("unique URLs to probe", "count", len(toProbe), "urls", toProbe)

	jobs := make(chan string)
	var wg sync.WaitGroup
	results := make(map[string]bool)
	var mu sync.Mutex

	for range lc.Workers {
		wg.Go(func() {
			for dedupedURL := range jobs {
				if ctx.Err() != nil {
					return
				}
				isInaccessible, _ := probeAccessibility(ctx, client, dedupedURL, lc)
				mu.Lock()
				results[dedupedURL] = isInaccessible
				mu.Unlock()
			}
		})
	}

send:
	for _, u := range toProbe {
		select {
		case <-ctx.Done(): // context canceled early exit
			break send
		case jobs <- u:
		}
	}
	close(jobs)
	wg.Wait()

	var inaccessibleCount int
	for _, u := range navigableURLs {
		key := dedupeKey(u)
		if inaccessible, ok := results[key]; ok && inaccessible {
			inaccessibleCount++
		}
	}

	return inaccessibleCount, nil
}

func deduplicated(urls []*url.URL) []string {
	seen := make(map[string]struct{})
	for _, u := range urls {
		seen[dedupeKey(u)] = struct{}{}
	}
	deduped := slices.Collect(maps.Keys(seen))
	return deduped
}

// dedupeKey returns the string representation of the URL without the fragment.
func dedupeKey(u *url.URL) string {
	nu := *u
	nu.Fragment = ""
	return nu.String()
}
