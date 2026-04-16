package analyzer

import (
	htmlstd "html"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

func titleFromDocument(doc *goquery.Document) string {
	// First <title> in tree
	t := strings.TrimSpace(htmlstd.UnescapeString(doc.Find("title").First().Text()))
	return t
}
