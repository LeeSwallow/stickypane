package app

import (
	"strings"

	"github.com/LeeSwallow/stickypane/internal/doc"
	"github.com/LeeSwallow/stickypane/internal/widget"
)

// book is the widget of a folder: its pages drawn one after another, each
// under a rule with its name, so that a book scrolls like one long note and
// every page is reached the same way everything else is. Keys go to the
// page at the top of the view; "," and "." jump to the page before or
// after it.
type book struct {
	pages  []page
	at     int   // the page the view is on
	starts []int // the line each page started at in the last Draw
}

func (b *book) current() page { return b.pages[max(min(b.at, len(b.pages)-1), 0)] }

// pageAt returns the page that holds line of the last Draw.
func (b *book) pageAt(line int) int {
	at := 0
	for i, s := range b.starts {
		if s <= line {
			at = i
		}
	}
	return at
}

// settle decides which page the view is on after it moved to offset with
// rows lines in view: the page it was on, as long as that page's start is
// still in view (a short book cannot scroll a late page to the top), else
// the page at the top of the view.
func (b *book) settle(offset, rows int) {
	if b.at < len(b.starts) {
		if s := b.starts[b.at]; s >= offset && s < offset+rows {
			return
		}
	}
	b.at = b.pageAt(offset)
}

// Draw implements widget.Widget.
func (b *book) Draw(width int, active bool) (string, widget.Span) {
	var out []string
	at := widget.NoSpan
	b.starts = b.starts[:0]
	for i, p := range b.pages {
		b.starts = append(b.starts, len(out))
		name := nameOf(p.note)
		rule := widget.Truncate(" "+name+" ", max(width-4, 1))
		if i == b.at {
			rule = widget.Accent.Render(rule)
		} else {
			rule = widget.Faint.Render(rule)
		}
		out = append(out, widget.Faint.Render("─")+rule+widget.Faint.Render(strings.Repeat("─", max(width-widget.Width(rule)-1, 0))))
		drawn, span := p.w.Draw(width, active && i == b.at)
		if span.Ok() && i == b.at {
			at = span.Shift(len(out))
		}
		out = append(out, strings.Split(drawn, "\n")...)
		if i < len(b.pages)-1 {
			out = append(out, "")
		}
	}
	return widget.Fit(strings.Join(out, "\n"), width), at
}

// Summary implements widget.Widget: which page, and what it counts.
func (b *book) Summary() string {
	s := b.current().w.Summary()
	at := strings.TrimSpace(strings.Join([]string{itoa(b.at + 1), "/", itoa(len(b.pages))}, ""))
	if s == "" {
		return at
	}
	return at + " · " + s
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var digits []byte
	for ; n > 0; n /= 10 {
		digits = append([]byte{byte('0' + n%10)}, digits...)
	}
	return string(digits)
}

// Update implements widget.Widget: the page the view is on takes the key.
func (b *book) Update(key string) (widget.Widget, widget.Result) {
	p := &b.pages[max(min(b.at, len(b.pages)-1), 0)]
	w, res := p.w.Update(key)
	p.w = w
	return b, res
}

// Click implements widget.Clicker: a click lands on the page it is in.
func (b *book) Click(line, col int) (widget.Widget, widget.Result, bool) {
	i := b.pageAt(line)
	c, ok := b.pages[i].w.(widget.Clicker)
	if !ok {
		return b, widget.Result{}, false
	}
	w, res, hit := c.Click(line-b.starts[i]-1, col)
	b.pages[i].w = w
	if hit {
		b.at = i
	}
	return b, res, hit
}

// Sync implements widget.Widget. A book is rebuilt from its files on every
// reload, so there is nothing to sync.
func (b *book) Sync(doc.Document) widget.Widget { return b }
