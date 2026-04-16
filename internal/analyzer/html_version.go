package analyzer

import (
	"bytes"
	"strings"

	"golang.org/x/net/html"
)

// detectHTMLVersion inspects the first doctype or early markup to classify the document.
func detectHTMLVersion(b []byte) (version, rawSnippet string) {
	z := html.NewTokenizer(bytes.NewReader(b))
	for {
		tt := z.Next()
		if tt == html.ErrorToken {
			return "Unknown", ""
		}
		if tt == html.DoctypeToken {
			raw := strings.TrimSpace(string(z.Raw()))
			if len(raw) > 160 {
				rawSnippet = raw[:160] + "…"
			} else {
				rawSnippet = raw
			}
			low := strings.ToLower(raw)
			// https://www.w3.org/TR/html4/
			if strings.Contains(low, "html 4") {
				return "HTML 4", rawSnippet
			}
			if strings.HasPrefix(low, "<!doctype html") || strings.Contains(low, "doctype html") {
				return "HTML5", rawSnippet
			}
			return "Unknown", rawSnippet
		}
		if tt == html.StartTagToken {
			return "Unknown", ""
		}
	}
}
