package link_checker

import (
	"net/url"
	"strings"
)

// sameHost reports whether target has the same hostname (domain) as page case-insensitively.
// for subdomain, here consider as false
func sameHost(page, target *url.URL) bool {
	if page == nil || target == nil {
		return false
	}
	if !strings.EqualFold(page.Hostname(), target.Hostname()) {
		return false
	}
	return true
}

// NavigableHTTP reports whether the URL uses http or https (eligible for probing).
func navigableHTTP(u *url.URL) bool {
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
