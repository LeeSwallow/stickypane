package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/LeeSwallow/stickypane/internal/doc"
	"github.com/LeeSwallow/stickypane/internal/store"
	"github.com/LeeSwallow/stickypane/internal/theme"
	"github.com/LeeSwallow/stickypane/internal/widget"
)

// counted is a widget that counts how often it is drawn.
type counted struct {
	draws *int
	text  string
}

func (c *counted) Draw(int, bool) (string, widget.Span) { *c.draws++; return c.text, widget.NoSpan }
func (c *counted) Summary() string                      { return "" }
func (c *counted) Update(key string) (widget.Widget, widget.Result) {
	c.text += key // changes in place, as real widgets do
	return c, widget.Result{}
}
func (c *counted) Sync(d doc.Document) widget.Widget { return &counted{draws: c.draws, text: d.Body} }

// A note that did not change is not drawn again when another key redraws
// the screen; one that took a key is.
func TestAnUnchangedNoteIsNotDrawnAgain(t *testing.T) {
	draws := 0
	kind := widget.Kind{Name: "note", Keys: "x", Parse: func(d doc.Document) widget.Widget { return &counted{draws: &draws, text: d.Body} }}
	dir := filepath.Join(t.TempDir(), store.DirName)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, dir, "a.md", opened("a"))
	writeFile(t, dir, "b.md", opened("b"))
	m := New(store.Open(dir), widget.Registry{kind}, nil, theme.NewHolder(theme.Pick("auto", true)))
	m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})

	draws = 0
	m.relayout()
	if draws != 0 {
		t.Errorf("nothing changed, yet %d notes were drawn", draws)
	}
	press(m, "x")
	if draws != 1 {
		t.Errorf("only the note that took the key is drawn again: %d draws", draws)
	}
	if s := screen(m); !strings.Contains(s, "ax") && !strings.Contains(s, "bx") {
		t.Errorf("the key shows on the screen:\n%s", s)
	}
}
