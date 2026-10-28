// Package html is the note that is a web page: HTML the overlay draws. In
// the terminal it is only its text, so it is view-only.
package html

import (
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
	Parse:    func(d doc.Document) widget.Widget { return parse(d) },
	Data:     data,
}
