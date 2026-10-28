package html

import (
	"strings"

	"github.com/LeeSwallow/stickypane/internal/core/doc"
	"github.com/LeeSwallow/stickypane/internal/core/widget"
)

// Draw implements widget.Widget: the text of the page, wrapped, then a line
// that says the page itself is drawn in the app. A page has no cursor.
func (p *Page) Draw(width int, _ bool) (string, widget.Span) {
	var out []string
	for _, line := range doc.SplitLines(widget.Clean(p.text)) {
		out = append(out, widget.Wrap(line, width)...)
	}
	if len(out) > 0 {
		out = append(out, "")
	}
	for _, l := range widget.Wrap(p.env.T("open it in the app to see it drawn"), width) {
		out = append(out, p.env.Styles.Faint.Render(l))
	}
	return strings.Join(out, "\n"), widget.NoSpan
}

// Summary implements widget.Widget: a page has none.
func (p *Page) Summary() string { return "" }

// Update implements widget.Widget. A page takes no keys.
func (p *Page) Update(string) (widget.Widget, widget.Result) { return p, widget.Result{} }

// Sync implements widget.Widget.
func (p *Page) Sync(d doc.Document) widget.Widget { return parse(d, p.env) }
