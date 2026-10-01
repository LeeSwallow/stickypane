// Package widget defines how one shape of note is drawn and edited. A widget
// knows nothing about the terminal framework or the file system: it receives
// keys as strings and answers with an intent to apply to the file.
package widget

import "github.com/LeeSwallow/stickypane/internal/doc"

// Widget draws one note and reacts to keys while the note is open.
type Widget interface {
	// Preview draws the note on the board. The kind decides how tall it is.
	Preview(width int) string
	// View draws the open note in exactly the given area or less.
	View(width, height int) string
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
	Name     string // the front matter "type" value
	Label    string // shown in the catalog
	FullRow  bool   // takes a whole row on the board
	Template func(title string) []byte
	Parse    func(d doc.Document) Widget
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
