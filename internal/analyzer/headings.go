package analyzer

import (
	"fmt"

	"github.com/PuerkitoBio/goquery"
)

func headingsFromDocument(doc *goquery.Document) [6]int {
	var counts [6]int
	for level := 1; level <= 6; level++ {
		sel := fmt.Sprintf("h%d", level)
		counts[level-1] = doc.Find(sel).Length()
	}
	return counts
}
