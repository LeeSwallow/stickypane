package html

import "github.com/LeeSwallow/stickypane/internal/doc"

// Page is what an html note says in the terminal: the text of it. The
// overlay draws the source, which data gives.
type Page struct {
	text string
}

func parse(d doc.Document) *Page { return &Page{text: text(d.Body)} }

// data is the source of the page, for the overlay to draw.
func data(d doc.Document) any {
	return struct {
		HTML string `json:"html"`
	}{d.Body}
}
