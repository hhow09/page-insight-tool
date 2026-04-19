package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLoad_emptyYAMLUsesDefaults(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "cfg.yaml")
	if err := os.WriteFile(p, []byte("# defaults\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	got, err := Load(p)
	if err != nil {
		t.Fatal(err)
	}
	want := Default()
	if got.Server.Addr != want.Server.Addr {
		t.Fatalf("server.addr: got %q want %q", got.Server.Addr, want.Server.Addr)
	}
	if err := got.Validate(); err != nil {
		t.Fatal(err)
	}
}

func TestLoad_partialOverride(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "app.yaml")
	content := `
server:
  addr: ":3000"
http_client:
  timeout: 2m
`
	if err := os.WriteFile(p, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	got, err := Load(p)
	if err != nil {
		t.Fatal(err)
	}
	if got.Server.Addr != ":3000" {
		t.Fatalf("addr: got %q", got.Server.Addr)
	}
	if got.HTTPClient.Timeout != 2*time.Minute {
		t.Fatalf("timeout: got %v", got.HTTPClient.Timeout)
	}
	if got.HTTPClient.MaxRedirects != Default().HTTPClient.MaxRedirects {
		t.Fatalf("max_redirects: got %d want %d", got.HTTPClient.MaxRedirects, Default().HTTPClient.MaxRedirects)
	}
	if err := got.Validate(); err != nil {
		t.Fatal(err)
	}
}

func TestLoad_partialHTTPClientPreservesOtherFields(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "app.yaml")
	if err := os.WriteFile(p, []byte("http_client:\n  timeout: 5s\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	got, err := Load(p)
	if err != nil {
		t.Fatal(err)
	}
	if got.HTTPClient.MaxRedirects != Default().HTTPClient.MaxRedirects {
		t.Fatalf("max_redirects overwritten: got %d", got.HTTPClient.MaxRedirects)
	}
	if got.HTTPClient.Timeout != 5*time.Second {
		t.Fatalf("timeout: got %v", got.HTTPClient.Timeout)
	}
}

func TestLoad_timeoutWrongYAMLType(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "bad.yaml")
	if err := os.WriteFile(p, []byte(`http_client:
  timeout: not-a-number
`), 0o600); err != nil {
		t.Fatal(err)
	}
	_, err := Load(p)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestLoad_timeoutNegativeFailsValidate(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "neg.yaml")
	if err := os.WriteFile(p, []byte(`http_client:
  timeout: -1s
`), 0o600); err != nil {
		t.Fatal(err)
	}
	got, err := Load(p)
	if err != nil {
		t.Fatal(err)
	}
	if err := got.Validate(); err == nil {
		t.Fatal("expected validate error for negative timeout")
	}
}

func TestLoad_noConfigFileUsesDefaults(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	got, err := Load("")
	if err != nil {
		t.Fatal(err)
	}
	want := Default()
	if got.Server.Addr != want.Server.Addr {
		t.Fatalf("server.addr: got %q want %q", got.Server.Addr, want.Server.Addr)
	}
	if err := got.Validate(); err != nil {
		t.Fatal(err)
	}
}

