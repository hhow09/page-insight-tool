package config

import (
	"testing"
)

func TestDefaultValidate(t *testing.T) {
	if err := Default().Validate(); err != nil {
		t.Fatal(err)
	}
}

func TestValidateHTTPClient(t *testing.T) {
	c := Default()
	c.HTTPClient.MaxRedirects = 0
	if err := c.Validate(); err == nil {
		t.Fatal("expected error for MaxRedirects=0")
	}
}

func TestValidateLinkWorkers(t *testing.T) {
	c := Default()
	c.Link.Workers = 0
	if err := c.Validate(); err == nil {
		t.Fatal("expected error for Workers=0")
	}
}
