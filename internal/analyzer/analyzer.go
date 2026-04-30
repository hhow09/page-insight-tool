package analyzer

import (
	"bytes"
	"context"
	"fmt"
	"io"

	"github.com/PuerkitoBio/goquery"
	"github.com/hhow09/page-insight-tool/internal/model"
)

// Analyze parses HTML from r and extracts title, version, heading counts, login signal, and raw anchor hrefs.
func Analyze(ctx context.Context, r io.Reader) (*model.AnalyzeReport, error) {
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
	rep := &model.AnalyzeReport{}
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
