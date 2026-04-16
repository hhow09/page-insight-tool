package fetch

import (
	"fmt"
	"net/http"

	"github.com/hhow09/page-insight-tool/internal/config"
)

func newHTTPClient(cfg *config.FetchConfig) *http.Client {
	return &http.Client{
		Timeout: cfg.Timeout,
		CheckRedirect: func(_ *http.Request, via []*http.Request) error {
			// Match net/http default policy shape: cap total redirect hops.
			if len(via) >= cfg.MaxRedirects {
				return fmt.Errorf("%w (limit %d)", ErrTooManyRedirects, cfg.MaxRedirects)
			}
			return nil
		},
	}
}
