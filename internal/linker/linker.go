package linker

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
	PartialCheck        bool
}

// Summarize resolves raw hrefs against page, counts internal vs external, then probes unique http(s) URLs.
func Summarize(ctx context.Context, client *http.Client, page *url.URL, rawHrefs []string, lc *config.LinkConfig) (*Summary, error) {
	summary, navigableURLs, err := collectNavigableLinks(page, rawHrefs)
	if err != nil {
		return nil, err
	}

	inAccessibleLinks, probePartial, err := runProbes(ctx, client, navigableURLs, lc)
	if probePartial {
		summary.PartialCheck = true
	}
	summary.InaccessibleLinks = inAccessibleLinks
	return summary, err
}

// collectNavigableLinks resolves each href, counts internal vs external and skips,
// and fills Summary with classification counts,
// and NavigableURLs (all resolved URLs to probe).
func collectNavigableLinks(page *url.URL, rawHrefs []string) (*Summary, []*url.URL, error) {
	if page == nil {
		return nil, nil, &url.Error{Op: "linker.collectNavigableLinks", URL: "", Err: errors.New("nil page URL")}
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
		navigable = append(navigable, abs)
	}

	return out, navigable, nil
}

// runProbes runs HEAD/GET accessibility checks for navigableURLs with a bounded worker pool, returning the total count of inaccessible links.
func runProbes(ctx context.Context, client *http.Client, navigableURLs []*url.URL, lc *config.LinkConfig) (inaccessible int, partial bool, err error) {
	if len(navigableURLs) == 0 {
		return 0, false, nil
	}
	toProbe, partial := dedupAndCap(navigableURLs, lc.MaxURLsToCheck)
	slog.Debug("unique URLs to probe", "count", len(toProbe), "urls", toProbe)

	jobs := make(chan string)
	var wg sync.WaitGroup
	results := make(map[string]bool)
	var mu sync.Mutex

	for range lc.Workers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for linkURL := range jobs {
				if ctx.Err() != nil {
					return
				}
				isInaccessible, _ := probeAccessibility(ctx, client, linkURL, lc)
				mu.Lock()
				results[linkURL] = isInaccessible
				mu.Unlock()
			}
		}()
	}

send:
	for _, u := range toProbe {
		select {
		case <-ctx.Done(): // context canceled early exit
			partial = true
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

	return inaccessibleCount, partial, nil
}

func dedupAndCap(urls []*url.URL, max int) ([]string, bool) {
	seen := make(map[string]struct{})
	for _, u := range urls {
		seen[dedupeKey(u)] = struct{}{}
	}
	deduped := slices.Collect(maps.Keys(seen))
	if max > 0 && len(deduped) >= max {
		return deduped[:max], true
	}
	return deduped, false
}

// dedupeKey returns the string representation of the URL without the fragment.
func dedupeKey(u *url.URL) string {
	nu := *u
	nu.Fragment = ""
	return nu.String()
}
