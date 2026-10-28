package html

import (
	"github.com/LeeSwallow/stickypane/internal/core/doc"
	"github.com/LeeSwallow/stickypane/internal/core/widget"
)

// Page is what an html note says in the terminal: the text of it. The
// overlay draws the source, which data gives.
type Page struct {
	text string
	env  widget.Env
}

func parse(d doc.Document, e widget.Env) *Page { return &Page{text: text(d.Body), env: e} }

// data is the source of the page, for the overlay to draw.
func data(d doc.Document, _ widget.Env) any {
	return struct {
		HTML string `json:"html"`
	}{d.Body}
}
