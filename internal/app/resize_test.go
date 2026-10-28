package app

import (
	"strings"
	"testing"
)

func TestHeightKeysWriteRows(t *testing.T) {
	m, dir := newModel(t, map[string]string{"a.md": opened("one\n")})
	writeView(t, dir, `{"notes":{"a.md":{"rows":5}}}`)
	m.reload()
	steps := []struct {
		key  string
		rows int
	}{
		{"shift+up", 6}, {"shift+down", 5}, {"shift+down", 4},
		{"shift+pgup", 9}, {"shift+pgdown", 4}, {"shift+pgdown", 1}, {"shift+down", 1},
		{"0", 0},
	}
	for _, step := range steps {
		press(m, step.key)
		if got := viewsOf(t, dir)["a.md"].Rows; got != step.rows {
			t.Fatalf("after %s: rows = %d, want %d", step.key, got, step.rows)
		}
		if readFile(t, dir, "a.md") != opened("one\n") {
			t.Fatalf("the height goes to sticky.json, not the note: %q", readFile(t, dir, "a.md"))
		}
	}
}

func TestHeightKeysStartFromWhatTheNoteShows(t *testing.T) {
	// A note that asks for nothing starts from its content; a note squeezed
	// by the others starts from what its pane shows.
	m, dir := newModel(t, map[string]string{
		"a.md": opened(numbered(3)),
		"b.md": opened(numbered(40)),
		"c.md": opened(numbered(40)),
	})
	writeView(t, dir, `{"notes":{"a.md":{"size":"1"},"b.md":{"size":"1"},"c.md":{"size":"1"}}}`)
	m.reload()
	m.relayout()
	a, _ := m.paneOf("a.md")
	press(m, "shift+up")
	if got := viewsOf(t, dir)["a.md"].Rows; got != a.total+1 {
		t.Errorf("a short note grows from its content (%d lines): rows = %d", a.total, got)
	}
	press(m, "tab")
	b, _ := m.paneOf("b.md")
	press(m, "shift+down")
	if got := viewsOf(t, dir)["b.md"].Rows; got != b.rows()-1 {
		t.Errorf("a squeezed note shrinks from what it shows (%d lines): rows = %d", b.rows(), got)
	}
}

func TestHeightKeysDoNothingOnAClosedNote(t *testing.T) {
	m, dir := newModel(t, map[string]string{"a.md": "closed\n"})
	press(m, "shift+up", "shift+down", "0")
	if v, ok := viewsOf(t, dir)["a.md"]; ok && v.Rows != 0 {
		t.Errorf("a closed note shows no height: %+v", v)
	}
}

func TestTheBorderShowsTheNewSizeForAMoment(t *testing.T) {
	m, _ := newModel(t, map[string]string{"a.md": opened("one\n")})
	press(m, "+")
	if top := bodyLines(m)[0]; !strings.Contains(top, "width 2/3") {
		t.Errorf("the border should say the new width: %q", top)
	}
	press(m, "shift+up")
	if top := bodyLines(m)[0]; !strings.Contains(top, "2 lines") || strings.Contains(top, "width") {
		t.Errorf("the border should say the new height: %q", top)
	}
	press(m, "j")
	if top := bodyLines(m)[0]; strings.Contains(top, "lines") {
		t.Errorf("the next key takes it away: %q", top)
	}
}
