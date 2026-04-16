package linker

import (
	"net/url"
	"strings"
)

// SameHost reports whether target has the same hostname (domain) as page case-insensitively.
// for subdomain, here consider as false
func SameHost(page, target *url.URL) bool {
	if page == nil || target == nil {
		return false
	}
	if !strings.EqualFold(page.Hostname(), target.Hostname()) {
		return false
	}
	return true
}

// NavigableHTTP reports whether the URL uses http or https (eligible for probing).
func NavigableHTTP(u *url.URL) bool {
	if u == nil {
		return false
	}
	switch strings.ToLower(u.Scheme) {
	case "http", "https":
		return true
	default:
		return false
	}
}
