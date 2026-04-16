package analyzer

import (
	"strings"

	"github.com/PuerkitoBio/goquery"
)

// rawHrefsFromDocument returns every `href` from `<a href>` in document order (may repeat).
func rawHrefsFromDocument(doc *goquery.Document) []string {
	var hrefs []string
	doc.Find("a[href]").Each(func(_ int, s *goquery.Selection) {
		href, ok := s.Attr("href")
		if !ok {
			return
		}
		href = strings.TrimSpace(href)
		if href == "" {
			return
		}
		hrefs = append(hrefs, href)
	})
	return hrefs
}
