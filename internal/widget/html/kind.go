// Package html is the note that is a web page: HTML the overlay draws. In
// the terminal it is only its text, so it is view-only.
package html

import (
	stdhtml "html"
	"strings"

	"github.com/LeeSwallow/stickypane/internal/doc"
	"github.com/LeeSwallow/stickypane/internal/widget"
)

// Kind registers the page. It is the kind of every ".html" and ".htm" file,
// and has no Act and no Events: nothing in a page can be changed from a screen.
var Kind = widget.Kind{
	Name:     "html",
	Exts:     []string{".html", ".htm"},
	New:      ".html",
	Label:    "HTML (view only)",
	Icon:     "⌗",
	Blurb:    "A web page. The terminal shows its text; the overlay draws it.",
	Command:  "stickypane show report.html",
	Usage:    "An .html or .htm file. The terminal shows the text of the page, without tags, scripts or styles; the desktop overlay draws the page itself. Nothing on it can be changed from a screen.",
	Example:  "<h1>Benchmark</h1>\n<p>Run took 4.2s, down from 6.1s.</p>\n",
	Size:     func(doc.Document) string { return widget.SizeHalf },
	Template: func(string) []byte { return []byte("<!doctype html>\n<title></title>\n") },
	Options: []widget.Option{
		{Name: "text", Type: widget.OptLines, Label: "Text"},
	},
	Compose: compose,
	Parse:   func(d doc.Document, e widget.Env) widget.Widget { return parse(d, e) },
	Data:    data,
}

// compose writes a new page: the title as its title and heading, and each
// line of text as a paragraph, all of it escaped, so what was typed is
// shown and never run.
func compose(title string, opts map[string]string) ([]byte, error) {
	var b strings.Builder
	b.WriteString("<!doctype html>\n<title>" + stdhtml.EscapeString(title) + "</title>\n")
	if strings.TrimSpace(title) != "" {
		b.WriteString("<h1>" + stdhtml.EscapeString(title) + "</h1>\n")
	}
	for _, l := range widget.Lines(opts["text"]) {
		b.WriteString("<p>" + stdhtml.EscapeString(l) + "</p>\n")
	}
	return []byte(b.String()), nil
}
