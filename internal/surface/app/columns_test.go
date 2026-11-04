package app

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
)

// order is the open notes' names in the order of the screen.
func order(m *Model) string {
	var names []string
	for _, p := range m.panes {
		names = append(names, strings.TrimSuffix(p.name, ".md"))
	}
	return strings.Join(names, " ")
}

func fiveNotes() map[string]string {
	files := map[string]string{}
	for _, n := range []string{"a", "b", "c", "d", "e"} {
		files[n+".md"] = opened(n + " note\n")
	}
	return files
}

// A board has three columns unless it is set otherwise: a note without a
// size takes one, and a row that is not full is shared equally.
func TestTheBoardIsThreeColumnsByDefault(t *testing.T) {
	t.Parallel()
	m, _ := newModel(t, fiveNotes())
	m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	want := map[string][3]int{"a": {0, 0, 40}, "b": {0, 40, 40}, "c": {0, 80, 40}, "d": {1, 0, 60}, "e": {1, 60, 60}}
	for i, p := range m.panes {
		name := strings.TrimSuffix(p.name, ".md")
		if got := [3]int{m.cells[i].Row, p.rect.X, p.rect.W}; got != want[name] {
			t.Errorf("%s: row, x, width = %v, want %v", name, got, want[name])
		}
	}
}

func TestASizedNoteSpansItsShareOfTheColumns(t *testing.T) {
	t.Parallel()
	m, dir := newModel(t, fiveNotes())
	writeView(t, dir, `{"columns":"4","notes":{"a.md":{"size":"1/2"},"c.md":{"size":"1"}}}`)
	m.Update(tea.WindowSizeMsg{Width: 120, Height: 60})
	m.reload()
	m.relayout()
	want := map[string][3]int{"a": {0, 0, 60}, "b": {0, 60, 60}, "c": {1, 0, 120}, "d": {2, 0, 60}, "e": {2, 60, 60}}
	for i, p := range m.panes {
		name := strings.TrimSuffix(p.name, ".md")
		if got := [3]int{m.cells[i].Row, p.rect.X, p.rect.W}; got != want[name] {
			t.Errorf("%s: row, x, width = %v, want %v", name, got, want[name])
		}
	}
}

// On a screen too narrow for the columns, fewer are drawn; nothing is
// written.
func TestANarrowScreenDrawsFewerColumns(t *testing.T) {
	t.Parallel()
	m, dir := newModel(t, fiveNotes())
	m.Update(tea.WindowSizeMsg{Width: 50, Height: 60})
	if m.grid() != 2 || m.panes[1].rect.Y != 0 || m.panes[2].rect.X != 0 {
		t.Errorf("50 cells hold two columns of at least 20: %d\n%s", m.grid(), screen(m))
	}
	if v := viewsOf(t, dir); len(v) != 0 {
		t.Errorf("a narrow screen writes nothing: %v", v)
	}
}

func TestAWidthStepIsOneColumn(t *testing.T) {
	t.Parallel()
	m, dir := newModel(t, map[string]string{"a.md": opened("one\n")})
	m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	for _, step := range []struct{ key, size string }{
		{"-", ""}, {"+", "2/3"}, {"+", "1"}, {"+", "1"}, {"-", "2/3"}, {"-", "1/3"}, {"-", "1/3"},
	} {
		press(m, step.key)
		if got := viewsOf(t, dir)["a.md"].Size; got != step.size {
			t.Fatalf("after %s: size = %q, want %q", step.key, got, step.size)
		}
	}
}

// H and L move a note a place, K and J a row.
func TestKeysMoveANoteAlongAndBetweenRows(t *testing.T) {
	t.Parallel()
	m, _ := newModel(t, fiveNotes())
	m.Update(tea.WindowSizeMsg{Width: 120, Height: 60})
	m.setFocus("b.md")
	for _, step := range []struct{ key, want string }{
		{"J", "a e c d b"}, {"K", "a b c d e"}, {"L", "a c b d e"}, {"H", "a b c d e"}, {"K", "a b c d e"},
	} {
		press(m, step.key)
		if got := order(m); got != step.want {
			t.Fatalf("after %s: %q, want %q", step.key, got, step.want)
		}
		if m.focus != "b.md" {
			t.Fatalf("after %s the focus moved to %q", step.key, m.focus)
		}
	}
	m.setFocus("a.md")
	press(m, "J")
	if got := order(m); got != "d b c a e" {
		t.Errorf("a moves to the note below it: %q", got)
	}
}

// On the board h j k l move between notes and H J K L move the note, a
// kanban too; the arrows reach its cards. Zoomed, a kanban takes h j k l
// and H J K L for its cards.
func TestAKanbanTakesHJKLWhenZoomed(t *testing.T) {
	t.Parallel()
	m, dir := newModel(t, map[string]string{"a.md": opened("one\n"), "b.md": boardFile})
	m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	m.setFocus("b.md")
	press(m, "H")
	if got := order(m); got != "b a" || !strings.Contains(readFile(t, dir, "b.md"), "## To do\n- payments") {
		t.Errorf("H moves the kanban on the board, not its card: %q\n%s", got, readFile(t, dir, "b.md"))
	}
	press(m, "l")
	if m.focus != "a.md" {
		t.Fatalf("l goes to the note on the right: %q", m.focus)
	}
	press(m, "h", "right")
	if m.focus != "b.md" || !strings.Contains(screen(m), "› login API") {
		t.Errorf("→ moves the kanban's cursor on the board:\n%s", screen(m))
	}
	press(m, "enter", "h", "L")
	if got := plain(readFile(t, dir, "b.md")); m.mode != modeZoom || !strings.Contains(got, "## To do\n## Doing\n- login API\n- payments") {
		t.Errorf("zoomed, h and L are the kanban's: %v\n%s", m.mode, got)
	}
	if got := order(m); got != "b a" {
		t.Errorf("the kanban's keys leave the note where it is: %q", got)
	}
}
