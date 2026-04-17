// Package fetch performs bounded HTTP GETs for the primary page request.
package fetch

import (
	"context"
	"fmt"
	"io"
	"log"
	"mime"
	"net/http"
	"net/url"
	"strings"

	"github.com/hhow09/page-insight-tool/internal/config"
)

const htmlContentType = "text/html"

// Result holds a successful fetch.
type Result struct {
	FinalURL *url.URL
	Body     []byte
	Status   int
}

// Fetch retrieves the URL body. Use [context.Context] for an overall deadline.
// If cfg is nil, [config.Default] is used.
func Fetch(ctx context.Context, raw string, cfg *config.FetchConfig) (*Result, error) {
	u, err := url.Parse(raw)
	if err != nil {
		return nil, &FetchError{HTTPStatus: 0, Message: fmt.Sprintf("invalid URL: %v", err)}
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return nil, &FetchError{HTTPStatus: 0, Message: "only http and https URLs are supported"}
	}
	if u.Host == "" {
		return nil, &FetchError{HTTPStatus: 0, Message: "URL is missing a host"}
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, &FetchError{HTTPStatus: 0, Message: fmt.Sprintf("build request: %v", err)}
	}
	req.Header.Set("User-Agent", cfg.UserAgent)

	client := newHTTPClient(cfg)
	resp, err := client.Do(req)
	if err != nil {
		return nil, mapDoError(err)
	}
	defer func() {
		if cerr := resp.Body.Close(); cerr != nil {
			log.Printf("fetch: close body: %v", cerr)
		}
	}()
	// Require a 2xx status.
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 8192))
		msg := http.StatusText(resp.StatusCode)
		if msg == "" {
			msg = fmt.Sprintf("HTTP status %d", resp.StatusCode)
		}
		return nil, &FetchError{HTTPStatus: resp.StatusCode, Message: msg}
	}

	// verify content type is HTML (except 204 which has no body/headers)
	if resp.StatusCode != http.StatusNoContent && !isHTML(resp) {
		return nil, &FetchError{HTTPStatus: 0, Message: "Only HTML is supported"}
	}

	limited := http.MaxBytesReader(nil, resp.Body, cfg.MaxBodyBytes)
	body, err := io.ReadAll(limited)
	if err != nil {
		return nil, &FetchError{HTTPStatus: resp.StatusCode, Message: fmt.Sprintf("read body: %v", err)}
	}

	final := resp.Request.URL
	return &Result{FinalURL: final, Body: body, Status: resp.StatusCode}, nil
}

func isHTML(resp *http.Response) bool {
	if s := strings.TrimSpace(resp.Header.Get("Content-Type")); s != "" {
		t, _, err := mime.ParseMediaType(s)
		if err != nil {
			return false
		}
		return t == htmlContentType
	}
	return false
}
