// Package widget defines how one shape of note is drawn and edited. A widget
// knows nothing about the terminal framework or the file system: it receives
// keys as strings and answers with an intent to apply to the file.
package widget

import (
	"path/filepath"
	"strings"
	"time"

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

// Clicker is implemented by widgets that can be worked with the mouse. Click
// is told which line and cell of the widget's last Draw was pressed. It
// reports whether anything was there; a press on plain text is not a hit.
type Clicker interface {
	Click(line, col int) (w Widget, res Result, hit bool)
}

// Result is what a key press asks the app to do. Every field may be empty.
type Result struct {
	Op     doc.Op  // apply to the file now
	Prompt *Prompt // collect one line of text first
	Run    bool    // run the note's file as a script, after asking the user
	Part   int     // with Run, which part of the note: the request of a .http file
}

// Prompt asks the app for a line of text and turns it into an Op.
type Prompt struct {
	Label   string
	Initial string // the text the line starts with
	Empty   bool   // an empty line is an answer too, not a cancel
	Submit  func(text string) doc.Op
}

// Kind registers one shape of note.
type Kind struct {
	Name string // the front matter "type" value
	// Exts are the file extensions, such as ".log", whose files are this
	// kind whatever they contain. Such files have no front matter.
	Exts []string
	// New is the extension of a file made as this kind, when it is not
	// Markdown. Such a file gets no front matter.
	New   string
	Label string // shown in the catalog
	Icon  string // one cell, shown before the note's title
	// Hint names the keys of Keys for the bottom line, as alternating
	// keys and what they do: {"h l", "column", "n", "new card"}.
	Hint []string
	// Blurb says in one line what this shape is for, and Example is a small
	// note of this shape; the catalog shows both.
	Blurb   string
	Example string
	// Command is the one line that makes or changes a note of this shape
	// from the command line, and Usage says in a sentence or two how the
	// note behaves. `stickypane kinds` prints them; the agent reads them.
	Command string
	Usage   string
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
	// Tend, when set, returns what stickypane writes into a note of this
	// kind by itself after the note changed from before to after at the
	// time changed, or nil when nothing is missing: the time an item was
	// ticked or a card moved by hand, say. before is empty when it is not
	// known. The board and the command line call it, so an agent never
	// has to.
	Tend func(before, after doc.Document, changed time.Time) doc.Op
	// Events, when set, says what happened in a note of this kind between
	// before and after, in the kind's own words: an item ticked, a card
	// moved. `stickypane watch` and the MCP server hand them to whatever
	// reacts to the board.
	Events func(before, after doc.Document) []Event
}

// Event is one thing that happened in a note: Type names it, such as
// "item.ticked" or "card.moved"; Item is what it happened to; From and To
// say what changed where that has two sides (a column, a value).
type Event struct {
	Type string `json:"type"`
	Item string `json:"item,omitempty"`
	From string `json:"from,omitempty"`
	To   string `json:"to,omitempty"`
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
		// A kind that is a file of its own, such as a script, cannot be
		// asked for by a Markdown note's front matter.
		if k.Name == name && k.New == "" {
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

// For returns the kind of the note in the file called name: the kind that
// owns the file's extension, or else the kind its front matter names.
func (r Registry) For(name string, d doc.Document) Kind {
	ext := strings.ToLower(filepath.Ext(name))
	for _, k := range r {
		for _, e := range k.Exts {
			if e == ext {
				return k
			}
		}
	}
	return r.Lookup(d.Type())
}

// Timed is implemented by widgets whose drawing changes as time passes, not
// only when their file does: a card that stalls after half an hour. The
// screen redraws at the time NextChange gives, and only then; the zero time
// means nothing is waiting.
type Timed interface {
	NextChange(now time.Time) time.Time
}

// Now is the clock every widget reads. The screen sets it to its own, so
// that what a widget draws and when the screen wakes agree.
var Now = time.Now
