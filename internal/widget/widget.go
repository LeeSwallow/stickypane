// Package widget defines how one shape of note is drawn and edited. A widget
// knows nothing about the terminal framework or the file system: it receives
// keys as strings and answers with an intent to apply to the file.
package widget

import (
	"strings"

	"github.com/LeeSwallow/stickypane/internal/doc"
)

// Sizes an open note can take on the screen.
const (
	SizePage = "page" // the whole width
	SizeHalf = "half" // half the width
	SizeCard = "card" // a small sticky note
)

// Span is a range of lines [Start, End) of a drawn note: the selection and
// everything that belongs to it, which the screen keeps in view.
type Span struct{ Start, End int }

// NoSpan is what a widget without a cursor returns.
var NoSpan = Span{Start: -1, End: -1}

// Ok reports whether s is a real span.
func (s Span) Ok() bool { return s.Start >= 0 }

// Shift moves a span down by n lines. NoSpan stays NoSpan.
func (s Span) Shift(n int) Span {
	if !s.Ok() {
		return s
	}
	return Span{Start: s.Start + n, End: s.End + n}
}

// Widget draws one note and reacts to keys while the note has the focus.
type Widget interface {
	// Draw renders the whole note at width. Nothing is cut: long text
	// wraps. When active, the widget shows its cursor and returns the lines
	// of the selection, including what unfolds under it; a widget without a
	// cursor, or one that is not active, returns NoSpan.
	Draw(width int, active bool) (out string, at Span)
	// Summary is a few words about the note for its border, such as
	// "2/3" or "5 cards". It may be empty.
	Summary() string
	// Update handles a key such as "j", "space" or "H".
	Update(key string) (Widget, Result)
	// Sync replaces the content after the file changed and keeps screen
	// state such as the cursor, clamped to the new content.
	Sync(d doc.Document) Widget
}

// Result is what a key press asks the app to do. Both fields may be nil.
type Result struct {
	Op     doc.Op  // apply to the file now
	Prompt *Prompt // collect one line of text first
}

// Prompt asks the app for a line of text and turns it into an Op.
type Prompt struct {
	Label  string
	Submit func(text string) doc.Op
}

// Kind registers one shape of note.
type Kind struct {
	Name  string // the front matter "type" value
	Label string // shown in the catalog
	Icon  string // one cell, shown before the note's title
	// Hint names the keys of Keys for the bottom line, as alternating
	// keys and what they do: {"h l", "column", "n", "new card"}.
	Hint []string
	// Blurb says in one line what this shape is for, and Example is a small
	// note of this shape; the catalog shows both.
	Blurb   string
	Example string
	// Keys lists, separated by spaces, the keys an open note of this kind
	// handles itself. Every other key belongs to the screen.
	Keys string
	// Tail makes a fixed-height view show the end of the note, as a log does.
	Tail bool
	// Rows is the default fixed height in lines. Zero means as tall as the
	// content.
	Rows int
	// Size returns the default size for a note: SizePage, SizeHalf or SizeCard.
	Size     func(d doc.Document) string
	Template func(title string) []byte
	Parse    func(d doc.Document) Widget
}

// Handles reports whether an open note of this kind takes the key.
func (k Kind) Handles(key string) bool {
	for _, own := range strings.Fields(k.Keys) {
		if own == key {
			return true
		}
	}
	return false
}

// Registry lists the known kinds. The first one is the fallback and must exist.
type Registry []Kind

// Lookup returns the kind for a type value. Unknown values get the first kind.
func (r Registry) Lookup(name string) Kind {
	for _, k := range r {
		if k.Name == name {
			return k
		}
	}
	return r[0]
}

// NewFile builds the content of a new note file.
func NewFile(kind, title, body string) []byte {
	d := doc.Document{Body: body}
	if kind != "" && kind != "note" {
		d = d.Set("type", kind)
	}
	if title != "" {
		d = d.Set("title", title)
	}
	return d.Bytes()
}
