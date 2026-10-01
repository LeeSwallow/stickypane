package app

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/LeeSwallow/stickypane/internal/doc"
)

// find returns the cell where text starts on the screen.
func find(t *testing.T, m *Model, text string) (x, y int) {
	t.Helper()
	for y, line := range strings.Split(screen(m), "\n") {
		if i := strings.Index(line, text); i >= 0 {
			return ansi.StringWidth(line[:i]), y
		}
	}
	t.Fatalf("%q is not on the screen:\n%s", text, screen(m))
	return 0, 0
}

func click(m *Model, x, y int) {
	m.Update(tea.MouseClickMsg{X: x, Y: y, Button: tea.MouseLeft})
	m.Update(tea.MouseReleaseMsg{X: x, Y: y, Button: tea.MouseLeft})
}

func clickOn(t *testing.T, m *Model, text string) {
	t.Helper()
	x, y := find(t, m, text)
	click(m, x, y)
}

func wheel(m *Model, x, y int, down bool) {
	b := tea.MouseWheelUp
	if down {
		b = tea.MouseWheelDown
	}
	m.Update(tea.MouseWheelMsg{X: x, Y: y, Button: b})
}

func TestTheScreenAsksForTheMouse(t *testing.T) {
	m, _ := newModel(t, nil)
	if got := m.View().MouseMode; got != tea.MouseModeCellMotion {
		t.Errorf("MouseMode = %v", got)
	}
}

func TestClickingATabOpensFocusesAndCloses(t *testing.T) {
	m, dir := newModel(t, map[string]string{
		"a.md": opened("first note\n"),
		"b.md": "---\ntitle: Second\n---\nsecond note\n",
	})
	clickOn(t, m, "Second")
	if v, _ := doc.Parse([]byte(readFile(t, dir, "b.md"))).Get("open"); v != "true" || m.focus != "b.md" {
		t.Fatalf("clicking a closed note's tab should open and focus it: open = %q, focus = %q", v, m.focus)
	}
	clickOn(t, m, "✎ a")
	if v, _ := doc.Parse([]byte(readFile(t, dir, "a.md"))).Get("open"); v != "true" || m.focus != "a.md" {
		t.Fatalf("clicking another open note's tab should only focus it: open = %q, focus = %q", v, m.focus)
	}
	clickOn(t, m, "✎ a")
	if v, _ := doc.Parse([]byte(readFile(t, dir, "a.md"))).Get("open"); v != "false" {
		t.Errorf("clicking the focused note's tab should close it: open = %q", v)
	}
}

func TestClickingANoteFocusesIt(t *testing.T) {
	m, _ := newModel(t, map[string]string{
		"a.md": opened("first note\n"),
		"b.md": opened("second note\n"),
	})
	if m.focus != "a.md" {
		t.Fatalf("focus = %q", m.focus)
	}
	clickOn(t, m, "second note")
	if m.focus != "b.md" || m.mode != modeBoard {
		t.Errorf("focus = %q, mode = %v", m.focus, m.mode)
	}
}

func TestClickingAChecklistItemTicksIt(t *testing.T) {
	m, dir := newModel(t, map[string]string{"a.md": opened("first\n"), "c.md": checklistFile})
	clickOn(t, m, "ship")
	if got := readFile(t, dir, "c.md"); !strings.HasSuffix(got, "- [x] build\n- [x] ship\n") {
		t.Errorf("a click on an item should tick it:\n%s", got)
	}
	if m.focus != "c.md" {
		t.Errorf("the clicked note should have the focus, got %q", m.focus)
	}
	clickOn(t, m, "build")
	if got := readFile(t, dir, "c.md"); !strings.HasSuffix(got, "- [ ] build\n- [x] ship\n") {
		t.Errorf("a click on a ticked item should untick it:\n%s", got)
	}
}

func TestClickingAFormAnswersIt(t *testing.T) {
	m, dir := newModel(t, map[string]string{"f.md": formFile})
	clickOn(t, m, "production")
	if got := readFile(t, dir, "f.md"); !strings.Contains(got, "- (x) production") {
		t.Fatalf("a click on an option should choose it:\n%s", got)
	}
	clickOn(t, m, "Cancel")
	if v, _ := doc.Parse([]byte(readFile(t, dir, "f.md"))).Get("submitted"); v != "Cancel" {
		t.Fatalf("a click on a button should press that button, submitted = %q", v)
	}
	x, y := find(t, m, "┃ ✓ Cancel")
	click(m, x-4, y) // the button to its left
	if v, _ := doc.Parse([]byte(readFile(t, dir, "f.md"))).Get("submitted"); v != "Go" {
		t.Errorf("each button has its own cells, submitted = %q", v)
	}
	x, y = find(t, m, "## Note")
	click(m, x+2, y+2) // inside the field's box
	if m.mode != modeInput {
		t.Errorf("a click on a field should ask for its text, mode = %v", m.mode)
	}
}

func TestClickingACardSelectsIt(t *testing.T) {
	m, dir := newModel(t, map[string]string{"b.md": boardFile})
	before := readFile(t, dir, "b.md")
	clickOn(t, m, "schema")
	if s := screen(m); !strings.Contains(s, "› schema") {
		t.Errorf("the clicked card should be selected:\n%s", s)
	}
	clickOn(t, m, "login API")
	if s := screen(m); !strings.Contains(s, "› login API") || strings.Contains(s, "› schema") {
		t.Errorf("the selection should follow the click:\n%s", s)
	}
	if readFile(t, dir, "b.md") != before {
		t.Error("selecting a card must not change the file")
	}
}

func TestTheWheelScrollsTheScreen(t *testing.T) {
	m, _ := newModel(t, map[string]string{"a.md": opened(numbered(60))})
	wheel(m, 10, 10, true)
	if m.scroll == 0 || !strings.Contains(screen(m), "line 05") || strings.Contains(screen(m), "line 01") {
		t.Fatalf("wheel down should scroll, scroll = %d:\n%s", m.scroll, screen(m))
	}
	wheel(m, 10, 10, false)
	wheel(m, 10, 10, false)
	if m.scroll != 0 {
		t.Errorf("wheel up should scroll back to the top, scroll = %d", m.scroll)
	}
}

func TestADoubleClickZoomsAndTheWheelScrollsTheZoomedNote(t *testing.T) {
	m, _ := newModel(t, map[string]string{"a.md": opened(numbered(60))})
	x, y := find(t, m, "line 03")
	click(m, x, y)
	if m.mode != modeBoard {
		t.Fatalf("one click must not zoom, mode = %v", m.mode)
	}
	click(m, x, y)
	if m.mode != modeZoom {
		t.Fatalf("a double click should zoom, mode = %v", m.mode)
	}
	wheel(m, 10, 10, true)
	if m.zoomScroll == 0 {
		t.Errorf("the wheel should scroll the zoomed note")
	}
}

func TestADoubleClickOnAControlDoesNotZoom(t *testing.T) {
	m, dir := newModel(t, map[string]string{"c.md": checklistFile})
	x, y := find(t, m, "ship")
	click(m, x, y)
	click(m, x, y)
	if m.mode != modeBoard {
		t.Errorf("two clicks on an item tick and untick it, mode = %v", m.mode)
	}
	if got := readFile(t, dir, "c.md"); !strings.HasSuffix(got, "- [ ] ship\n") {
		t.Errorf("file = %q", got)
	}
}

func TestClicksWorkInAZoomedNote(t *testing.T) {
	m, dir := newModel(t, map[string]string{"c.md": checklistFile})
	press(m, "z")
	clickOn(t, m, "ship")
	if got := readFile(t, dir, "c.md"); !strings.HasSuffix(got, "- [x] ship\n") {
		t.Errorf("a click in a zoomed checklist should tick the item:\n%s", got)
	}
}

func TestTheMouseIsIgnoredWhileAsking(t *testing.T) {
	m, dir := newModel(t, map[string]string{"c.md": checklistFile})
	x, y := find(t, m, "ship")
	press(m, "N")
	click(m, x, y)
	wheel(m, x, y, true)
	if m.mode != modeInput {
		t.Errorf("a click should not end the prompt, mode = %v", m.mode)
	}
	if got := readFile(t, dir, "c.md"); !strings.HasSuffix(got, "- [ ] ship\n") {
		t.Errorf("a click behind a prompt must do nothing:\n%s", got)
	}
}

func TestClicksOutsideEverythingDoNothing(t *testing.T) {
	m, _ := newModel(t, map[string]string{"a.md": opened("note\n")})
	for _, p := range [][2]int{{79, 23}, {79, 10}, {0, 23}, {-1, -1}, {500, 500}} {
		click(m, p[0], p[1])
		wheel(m, p[0], p[1], true)
	}
	if m.focus != "a.md" || m.mode != modeBoard {
		t.Errorf("focus = %q, mode = %v", m.focus, m.mode)
	}
}
