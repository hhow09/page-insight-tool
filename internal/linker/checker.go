package linker

import (
	"context"
	"io"
	"net/http"

	"github.com/hhow09/page-insight-tool/internal/config"
)

// probeAccessibility returns true if the URL should count as inaccessible (network error, or HTTP status >= 400).
func probeAccessibility(ctx context.Context, client *http.Client, linkURL string, lc *config.LinkConfig) (inaccessible bool, status int) {
	pctx, cancel := context.WithTimeout(ctx, lc.PerLinkTimeout)
	defer cancel()
	status, err := probeHTTP(pctx, client, http.MethodHead, linkURL, lc)
	if err != nil {
		return true, 0
	}
	if status == http.StatusMethodNotAllowed || status == http.StatusNotImplemented {
		gctx, gcancel := context.WithTimeout(ctx, lc.PerLinkTimeout)
		defer gcancel()
		status, err = probeHTTP(gctx, client, http.MethodGet, linkURL, lc)
		if err != nil {
			return true, 0
		}
	}
	if status >= http.StatusBadRequest {
		return true, status
	}
	return false, status
}

// probeHTTP runs a single request, drains the body up to lc.MaxProbeBodyBytes, and returns the response status.
func probeHTTP(ctx context.Context, client *http.Client, method, linkURL string, lc *config.LinkConfig) (status int, err error) {
	req, err := http.NewRequestWithContext(ctx, method, linkURL, nil)
	if err != nil {
		return 0, err
	}
	req.Header.Set("User-Agent", lc.UserAgent)
	resp, err := client.Do(req)
	if err != nil {
		return 0, err
	}
	defer func() { _ = resp.Body.Close() }()
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, lc.MaxProbeBodyBytes))
	return resp.StatusCode, nil
}
