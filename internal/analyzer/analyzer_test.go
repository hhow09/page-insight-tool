package analyzer

import (
	"bytes"
	"context"
	_ "embed"
	"reflect"
	"strings"
	"testing"
)

//go:embed test_data/go_blog.html
var goBlogFixture []byte

//go:embed test_data/home24_login.html
var home24LoginFixture []byte

//go:embed test_data/amazon_login.html
var amazonLoginFixture []byte

func TestAnalyze_HTML5_title_headings_links(t *testing.T) {
	t.Parallel()
	html := `<!DOCTYPE html>
<html><head><title>  Hello &amp; Co  </title></head>
<body>
<h1>One</h1>
<h2></h2><h2>Two</h2>
<h3>x</h3><h3>y</h3><h3>z</h3>
<a href="/a">a</a>
<a href="https://ex.org/b">b</a>
<a href="/a">dup</a>
</body></html>`
	rep, err := Analyze(context.Background(), strings.NewReader(html))
	if err != nil {
		t.Fatal(err)
	}
	if rep.HTMLVersion != "HTML5" {
		t.Fatalf("HTMLVersion: got %q", rep.HTMLVersion)
	}
	if rep.Title != "Hello & Co" {
		t.Fatalf("Title: got %q", rep.Title)
	}
	want := [6]int{1, 2, 3, 0, 0, 0}
	if rep.HeadingsCount != want {
		t.Fatalf("Headings: got %v want %v", rep.HeadingsCount, want)
	}
	if rep.LoginForm {
		t.Fatal("LoginForm: want false")
	}
	wantHrefs := []string{"/a", "https://ex.org/b", "/a"}
	if !reflect.DeepEqual(rep.RawHrefs, wantHrefs) {
		t.Fatalf("RawHrefs: %#v", rep.RawHrefs)
	}
}

func TestAnalyze_malformedHTML(t *testing.T) {
	t.Parallel()
	html := `<!doctype html><html><head><title>T</title><h1>oops in head</h1>
<body><p>unclosed
<h1>ok</h1><a href="#x">x</a>`
	rep, err := Analyze(context.Background(), strings.NewReader(html))
	if err != nil {
		t.Fatal(err)
	}
	if rep.Title != "T" {
		t.Fatalf("Title: %q", rep.Title)
	}
	if rep.HeadingsCount[0] < 1 {
		t.Fatalf("expected at least one h1, got %v", rep.HeadingsCount)
	}
	if got := len(rep.RawHrefs); got != 1 {
		t.Fatalf("hrefs: got %d", got)
	}
}

func TestAnalyze_login_password(t *testing.T) {
	t.Parallel()
	html := `<!DOCTYPE html><html><head><title>x</title></head><body>
<form action="/login"><input name="u"><input type="PASSWORD" name="p"></form>
</body></html>`
	rep, err := Analyze(context.Background(), strings.NewReader(html))
	if err != nil {
		t.Fatal(err)
	}
	if !rep.LoginForm {
		t.Fatal("expected LoginForm true")
	}
}

func TestAnalyze_login_noForm(t *testing.T) {
	t.Parallel()
	html := `<html><body><input type="password" name="p"></body></html>`
	rep, err := Analyze(context.Background(), strings.NewReader(html))
	if err != nil {
		t.Fatal(err)
	}
	if rep.LoginForm {
		t.Fatal("password outside form should not count as login form for this detector")
	}
}

func TestAnalyze_version_unknownWhenNoDoctype(t *testing.T) {
	t.Parallel()
	html := `<html><head><title>x</title></head><body></body></html>`
	rep, err := Analyze(context.Background(), strings.NewReader(html))
	if err != nil {
		t.Fatal(err)
	}
	if rep.HTMLVersion != "Unknown" {
		t.Fatalf("got %q", rep.HTMLVersion)
	}
}

func TestAnalyze_go_blog_fixture(t *testing.T) {
	t.Parallel()
	rep, err := Analyze(context.Background(), bytes.NewReader(goBlogFixture))
	if err != nil {
		t.Fatal(err)
	}
	if rep.HTMLVersion != "HTML5" {
		t.Fatalf("HTMLVersion: got %q", rep.HTMLVersion)
	}
	const wantTitle = "The Go Blog - The Go Programming Language"
	if rep.Title != wantTitle {
		t.Fatalf("Title: got %q want %q", rep.Title, wantTitle)
	}
	wantHeadings := [6]int{1, 0, 0, 0, 0, 0}
	if rep.HeadingsCount != wantHeadings {
		t.Fatalf("Headings: got %v want %v", rep.HeadingsCount, wantHeadings)
	}
	if rep.LoginForm {
		t.Fatal("LoginForm: expected false for public blog listing")
	}
	const wantHrefs = 102
	if got := len(rep.RawHrefs); got != wantHrefs {
		t.Fatalf("RawHrefs: got %d want %d", got, wantHrefs)
	}
	// First navigational link in the fixture is the site root (header logo).
	if rep.RawHrefs[0] != "/" {
		t.Fatalf("RawHrefs[0]: got %q want %q", rep.RawHrefs[0], "/")
	}
	foundPkg := false
	for _, h := range rep.RawHrefs {
		if strings.Contains(h, "pkg.go.dev") {
			foundPkg = true
			break
		}
	}
	if !foundPkg {
		t.Fatal("expected at least one href to pkg.go.dev in fixture")
	}
}

func TestAnalyze_home24_login_fixture(t *testing.T) {
	t.Parallel()
	rep, err := Analyze(context.Background(), bytes.NewReader(home24LoginFixture))
	if err != nil {
		t.Fatal(err)
	}
	if rep.HTMLVersion != "HTML5" {
		t.Fatalf("HTMLVersion: got %q", rep.HTMLVersion)
	}
	const wantTitle = "Anmelden | home24"
	if rep.Title != wantTitle {
		t.Fatalf("Title: got %q want %q", rep.Title, wantTitle)
	}
	if !rep.LoginForm {
		t.Fatal("LoginForm: expected true (fixture contains <form> and <input type=\"password\">)")
	}
}

func TestAnalyze_amazon_login_fixture(t *testing.T) {
	t.Parallel()
	rep, err := Analyze(context.Background(), bytes.NewReader(amazonLoginFixture))
	if err != nil {
		t.Fatal(err)
	}
	if rep.HTMLVersion != "HTML5" {
		t.Fatalf("HTMLVersion: got %q", rep.HTMLVersion)
	}
	const wantTitle = "Amazon Sign-In"
	if rep.Title != wantTitle {
		t.Fatalf("Title: got %q want %q", rep.Title, wantTitle)
	}
	if !rep.LoginForm {
		t.Fatal("LoginForm: expected true (fixture contains sign-in form with password field)")
	}
}
