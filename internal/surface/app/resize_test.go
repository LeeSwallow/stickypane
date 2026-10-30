package app

import (
	"fmt"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
)

// drag presses the left button at one cell, moves to the others with it
// held, and lets go at the last.
func drag(m *Model, cells ...[2]int) {
	m.Update(tea.MouseClickMsg{X: cells[0][0], Y: cells[0][1], Button: tea.MouseLeft})
	for _, c := range cells[1:] {
		m.Update(tea.MouseMotionMsg{X: c[0], Y: c[1], Button: tea.MouseLeft})
	}
	last := cells[len(cells)-1]
	m.Update(tea.MouseReleaseMsg{X: last[0], Y: last[1], Button: tea.MouseLeft})
}

// edges returns the screen cells of a pane's right border and of its
// bottom border, each halfway along.
func edges(t *testing.T, m *Model, name string) (right, bottom [2]int) {
	t.Helper()
	p, ok := m.paneOf(name)
	if !ok {
		t.Fatalf("%s has no pane", name)
	}
	bar, r := len(m.bar), p.rect
	return [2]int{r.X + r.W - 1, bar + r.Y + r.H/2}, [2]int{r.X + r.W/2, bar + r.Y + r.H - 1}
}

func TestSixShortNotesFitOneScreen(t *testing.T) {
	t.Parallel()
	files := map[string]string{}
	for i := 1; i <= 6; i++ {
		files[fmt.Sprintf("n%d.md", i)] = opened(fmt.Sprintf("note %d\n", i) + numbered(5))
	}
	m, _ := newModel(t, files)
	t.Logf("80x24:\n%s", screen(m))
	if m.screens != 1 {
		t.Errorf("six notes of six lines fit one screen of 80x24, not %d:\n%s", m.screens, screen(m))
	}
	for i := 1; i <= 6; i++ {
		if !strings.Contains(screen(m), fmt.Sprintf("note %d", i)) {
			t.Errorf("note %d is not on the screen:\n%s", i, screen(m))
		}
	}
}

func TestTheRowsLeftoverWidthIsSharedByFraction(t *testing.T) {
	t.Parallel()
	m, dir := newModel(t, map[string]string{"a.md": opened("a\n"), "b.md": opened("b\n"), "c.md": opened("c\n")})
	writeView(t, dir, `{"notes":{"a.md":{"size":"1/3"},"b.md":{"size":"1/3"},"c.md":{"size":"1/3"}}}`)
	m.reload()
	m.relayout()
	var widths []int
	for _, n := range []string{"a.md", "b.md", "c.md"} {
		p, _ := m.paneOf(n)
		widths = append(widths, p.rect.W)
	}
	if fmt.Sprint(widths) != "[27 27 26]" {
		t.Errorf("three thirds of 80 cells are %v, want [27 27 26]: no note takes all of the rounding", widths)
	}
}

func TestDraggingTheRightBorderSnapsTheWidthToAFraction(t *testing.T) {
	t.Parallel()
	m, dir := newModel(t, map[string]string{"a.md": opened("one\n"), "b.md": opened("two\n")})
	writeView(t, dir, `{"notes":{"a.md":{"size":"1/2"},"b.md":{"size":"1/2"}}}`)
	m.reload()
	m.relayout()
	right, _ := edges(t, m, "a.md")
	m.Update(tea.MouseClickMsg{X: right[0], Y: right[1], Button: tea.MouseLeft})
	m.Update(tea.MouseMotionMsg{X: right[0] - 12, Y: right[1], Button: tea.MouseLeft})
	if got := viewsOf(t, dir)["a.md"].Size; got != "1/2" {
		t.Errorf("nothing is written while the border is held: %q", got)
	}
	if p, _ := m.paneOf("a.md"); p.rect.W >= 40 {
		t.Errorf("the screen follows the border as it moves: %+v", p.rect)
	}
	m.Update(tea.MouseReleaseMsg{X: right[0] - 12, Y: right[1], Button: tea.MouseLeft})
	if got := viewsOf(t, dir)["a.md"].Size; got != "1/3" {
		t.Errorf("a border let go at 28 cells snaps to 1/3, got %q", got)
	}
	if m.focus != "a.md" || m.mode != modeBoard {
		t.Errorf("focus = %q, mode = %v", m.focus, m.mode)
	}
	right, _ = edges(t, m, "a.md")
	drag(m, [2]int{right[0], right[1]}, [2]int{right[0] + 30, right[1]})
	if got := viewsOf(t, dir)["a.md"].Size; got != "3/4" && got != "2/3" {
		t.Errorf("dragged wider: %q", got)
	}
}

func TestDraggingTheBottomBorderSetsRows(t *testing.T) {
	t.Parallel()
	m, dir := newModel(t, map[string]string{"a.md": opened(numbered(40)), "b.md": opened(numbered(40))})
	writeView(t, dir, `{"notes":{"a.md":{"size":"1"},"b.md":{"size":"1"}}}`)
	m.reload()
	m.relayout()
	before, _ := m.paneOf("a.md")
	_, bottom := edges(t, m, "a.md")
	drag(m, bottom, [2]int{bottom[0], bottom[1] - 2}, [2]int{bottom[0], bottom[1] - 4})
	want := before.rect.H - 4 - 2
	if got := viewsOf(t, dir)["a.md"].Rows; got != want {
		t.Errorf("rows = %d, want %d: the pane is 4 lines shorter", got, want)
	}
	if p, _ := m.paneOf("a.md"); p.rect.H != before.rect.H-4 {
		t.Errorf("pane = %+v, was %+v", p.rect, before.rect)
	}
	if got := readFile(t, dir, "a.md"); got != opened(numbered(40)) {
		t.Errorf("the note is not touched: %q", got)
	}
}

func TestADoubleClickOnTheBottomBorderGoesBackToTheContent(t *testing.T) {
	t.Parallel()
	m, dir := newModel(t, map[string]string{"a.md": opened("one\n"), "b.md": opened("two\n")})
	writeView(t, dir, `{"notes":{"a.md":{"size":"1","rows":5},"b.md":{"size":"1"}}}`)
	m.reload()
	m.relayout()
	_, bottom := edges(t, m, "a.md")
	click(m, bottom[0], bottom[1])
	if got := viewsOf(t, dir)["a.md"].Rows; got != 5 {
		t.Fatalf("one click on a border changes nothing: rows = %d", got)
	}
	click(m, bottom[0], bottom[1])
	if got := viewsOf(t, dir)["a.md"].Rows; got != 0 || m.mode != modeBoard {
		t.Errorf("a double click on the bottom border should forget the height: rows = %d, mode = %v", got, m.mode)
	}
}

func TestHeightKeysWriteRows(t *testing.T) {
	t.Parallel()
	m, dir := newModel(t, map[string]string{"a.md": opened("one\n")})
	writeView(t, dir, `{"notes":{"a.md":{"rows":5}}}`)
	m.reload()
	m.relayout()
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
	t.Parallel()
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
	if b2, _ := m.paneOf("b.md"); b2.rect.H >= b.rect.H {
		t.Errorf("its pane should be shorter now: %d, was %d", b2.rect.H, b.rect.H)
	}
}

func TestHeightKeysDoNothingOnAClosedNote(t *testing.T) {
	t.Parallel()
	m, dir := newModel(t, map[string]string{"a.md": "closed\n"})
	press(m, "shift+up", "shift+down", "0")
	if v, ok := viewsOf(t, dir)["a.md"]; ok && v.Rows != 0 {
		t.Errorf("a closed note shows no height: %+v", v)
	}
}

func TestTheBorderShowsTheNewSizeForAMoment(t *testing.T) {
	t.Parallel()
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
