package html

import "github.com/LeeSwallow/stickypane/internal/doc"

// Page is what an html note says: its source, as the overlay draws it, and
// the text of it, as the terminal shows it.
type Page struct {
	source string
	text   string
}

func parse(d doc.Document) *Page { return &Page{source: d.Body, text: text(d.Body)} }

// data is the source of the page, for the overlay to draw.
func data(d doc.Document) any {
	return struct {
		HTML string `json:"html"`
	}{d.Body}
}
