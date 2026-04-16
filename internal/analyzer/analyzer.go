// Package analyzer extracts page insight metrics from HTML (Phase 3: parse only; no outbound link checks).
package analyzer

import (
	"bytes"
	"context"
	"fmt"
	"io"

	"github.com/PuerkitoBio/goquery"
)

// Report holds metrics extracted from a single HTML document.
type Report struct {
	HTMLVersion    string
	HTMLVersionRaw string
	Title          string // <title>
	HeadingsCount  [6]int // index 0 = h1 … index 5 = h6
	LoginForm      bool   // true if a password <input> appears inside a <form>
	RawHrefs       []string
}

// Analyze parses HTML from r and extracts title, version, heading counts, login signal, and raw anchor hrefs.
func Analyze(ctx context.Context, r io.Reader) (*Report, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	body, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("read html: %w", err)
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	rep := &Report{}
	rep.HTMLVersion, rep.HTMLVersionRaw = detectHTMLVersion(body)
	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("parse html: %w", err)
	}

	rep.Title = titleFromDocument(doc)
	rep.HeadingsCount = headingsFromDocument(doc)
	rep.LoginForm = loginFormFromDocument(doc)
	rep.RawHrefs = rawHrefsFromDocument(doc)
	return rep, nil
}
