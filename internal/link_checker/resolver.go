package link_checker

import (
	"fmt"
	"net/url"
)

// Resolve parses href as a reference against the page URL (RFC 3986).
func Resolve(page *url.URL, href string) (*url.URL, error) {
	if page == nil {
		return nil, fmt.Errorf("nil page URL")
	}
	return page.Parse(href)
}
