package link_checker

import (
	"context"
	"io"
	"net/http"
)

// probeAccessibility returns true if the URL should count as inaccessible (network error, or HTTP status >= 400).
func (lc *LinkChecker) probeAccessibility(ctx context.Context, linkURL string) (inaccessible bool, status int) {
	status, err := lc.probeHTTP(ctx, http.MethodHead, linkURL)
	if err != nil {
		return true, 0
	}
	if status == http.StatusMethodNotAllowed || status == http.StatusNotImplemented {
		status, err = lc.probeHTTP(ctx, http.MethodGet, linkURL)
		if err != nil {
			return true, 0
		}
	}
	if status >= http.StatusBadRequest {
		return true, status
	}
	return false, status
}

// probeHTTP runs a single request, drains the body, and returns the response status.
func (lc *LinkChecker) probeHTTP(ctx context.Context, method, linkURL string) (status int, err error) {
	req, err := http.NewRequestWithContext(ctx, method, linkURL, nil)
	if err != nil {
		return 0, err
	}
	resp, err := lc.client.Do(req)
	if err != nil {
		return 0, err
	}
	// always drain and close the body to allow connection reuse.
	defer func() { _ = resp.Body.Close() }()
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, lc.cfg.MaxProbeBodyBytes))
	return resp.StatusCode, nil
}
