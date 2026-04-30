package link_checker

import (
	"net/url"
	"testing"
)

func TestSameHost_differentSubdomain(t *testing.T) {
	t.Parallel()
	page, _ := url.Parse("https://www.example.com/page")
	sub, _ := url.Parse("https://blog.example.com/other")
	if sameHost(page, sub) {
		t.Fatal("www vs blog subdomain should not match")
	}
	apex, _ := url.Parse("https://example.com/")
	if sameHost(page, apex) {
		t.Fatal("www vs apex domain should not match")
	}
	if sameHost(apex, sub) {
		t.Fatal("apex vs subdomain should not match")
	}
	same, _ := url.Parse("https://WWW.example.com:443/path")
	if !sameHost(page, same) {
		t.Fatal("same logical host (case and default port) should match")
	}
}

func TestNavigableHTTP(t *testing.T) {
	t.Parallel()
	u, _ := url.Parse("mailto:x@y")
	if navigableHTTP(u) {
		t.Fatal("mailto not navigable")
	}
	h, _ := url.Parse("https://a.b/")
	if !navigableHTTP(h) {
		t.Fatal("https navigable")
	}
}
