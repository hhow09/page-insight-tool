package linker

import (
	"net/url"
	"testing"
)


func TestSameHost_differentSubdomain(t *testing.T) {
	t.Parallel()
	page, _ := url.Parse("https://www.example.com/page")
	sub, _ := url.Parse("https://blog.example.com/other")
	if SameHost(page, sub) {
		t.Fatal("www vs blog subdomain should not match")
	}
	apex, _ := url.Parse("https://example.com/")
	if SameHost(page, apex) {
		t.Fatal("www vs apex domain should not match")
	}
	if SameHost(apex, sub) {
		t.Fatal("apex vs subdomain should not match")
	}
	same, _ := url.Parse("https://WWW.example.com:443/path")
	if !SameHost(page, same) {
		t.Fatal("same logical host (case and default port) should match")
	}
}

func TestNavigableHTTP(t *testing.T) {
	t.Parallel()
	u, _ := url.Parse("mailto:x@y")
	if NavigableHTTP(u) {
		t.Fatal("mailto not navigable")
	}
	h, _ := url.Parse("https://a.b/")
	if !NavigableHTTP(h) {
		t.Fatal("https navigable")
	}
}
