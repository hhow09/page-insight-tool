package linker

import (
	"net/url"
	"testing"
)

func TestResolve_relative(t *testing.T) {
	t.Parallel()
	base, err := url.Parse("https://example.com/a/b/c.html")
	if err != nil {
		t.Fatal(err)
	}
	u, err := Resolve(base, "../d")
	if err != nil {
		t.Fatal(err)
	}
	if u.String() != "https://example.com/a/d" {
		t.Fatalf("got %s", u.String())
	}
}
