package analyzer

import (
	"strings"

	"github.com/PuerkitoBio/goquery"
)

// loginFormFromDocument reports whether any <form> contains an <input type="password">.
func loginFormFromDocument(doc *goquery.Document) bool {
	found := false
	doc.Find("form").Each(func(_ int, f *goquery.Selection) {
		f.Find("input").Each(func(_ int, inp *goquery.Selection) {
			typ, _ := inp.Attr("type")
			if strings.EqualFold(strings.TrimSpace(typ), "password") {
				found = true
			}
		})
	})
	return found
}
