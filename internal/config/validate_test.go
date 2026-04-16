package config

import (
	"testing"
	"time"
)

func TestDefaultValidate(t *testing.T) {
	if err := Default().Validate(); err != nil {
		t.Fatal(err)
	}
}

func TestValidateLinkWorkers(t *testing.T) {
	c := Default()
	c.Link.Workers = 0
	if err := c.Validate(); err == nil {
		t.Fatal("expected error for Workers=0")
	}
}

func TestValidateFetchTimeout(t *testing.T) {
	c := Default()
	c.Fetch.Timeout = 0
	if err := c.Validate(); err == nil {
		t.Fatal("expected error for Fetch.Timeout=0")
	}
}

func TestValidateLinkPerLinkTimeout(t *testing.T) {
	c := Default()
	c.Link.PerLinkTimeout = -time.Second
	if err := c.Validate(); err == nil {
		t.Fatal("expected error for negative PerLinkTimeout")
	}
}
