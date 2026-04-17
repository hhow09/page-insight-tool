package httpclient

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/hhow09/page-insight-tool/internal/config"
)

// ErrTooManyRedirects is returned when the HTTP redirect chain exceeds the configured limit.
var ErrTooManyRedirects = errors.New("too many redirects")

type userAgentTransport struct {
	userAgent string
	base      http.RoundTripper
}

func (t *userAgentTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if req.Header.Get("User-Agent") == "" && t.userAgent != "" {
		req.Header.Set("User-Agent", t.userAgent)
	}
	return t.base.RoundTrip(req)
}

func New(cfg *config.HTTPClientConfig) *http.Client {
	transport := http.DefaultTransport
	if cfg.UserAgent != "" {
		transport = &userAgentTransport{
			userAgent: cfg.UserAgent,
			base:      http.DefaultTransport,
		}
	}

	return &http.Client{
		Timeout:   cfg.Timeout,
		Transport: transport,
		CheckRedirect: func(_ *http.Request, via []*http.Request) error {
			// Match net/http default policy shape: cap total redirect hops.
			if len(via) >= cfg.MaxRedirects {
				return fmt.Errorf("%w (limit %d)", ErrTooManyRedirects, cfg.MaxRedirects)
			}
			return nil
		},
	}
}
