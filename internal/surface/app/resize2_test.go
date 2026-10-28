package app

import (
	"testing"

	"github.com/LeeSwallow/stickypane/internal/adapter/store"
)

func TestADragOfOneCellKeepsTheWidthOfAStretchedNote(t *testing.T) {
	t.Parallel()
	// 1/2 and 1/3 of 80 cells are 40 and 26, stretched to 48 and 32.
	m, dir := newModel(t, map[string]string{"a.md": opened("one\n"), "b.md": opened("two\n")})
	writeView(t, dir, `{"notes":{"a.md":{"size":"1/2"},"b.md":{"size":"1/3"}}}`)
	m.reload()
	m.relayout()
	for _, dx := range []int{-1, 1, -3, 3} {
		right, _ := edges(t, m, "a.md")
		drag(m, right, [2]int{right[0] + dx, right[1]})
		if got := viewsOf(t, dir)["a.md"].Size; got != "1/2" {
			t.Fatalf("a drag of %d cells on a stretched 1/2 wrote %q", dx, got)
		}
	}
}

func TestALoneStretchedNoteNeedsARealMoveToChange(t *testing.T) {
	t.Parallel()
	m, dir := newModel(t, map[string]string{"a.md": opened("one\n"), "b.md": opened("two\n")})
	writeView(t, dir, `{"notes":{"a.md":{"size":"1/2"},"b.md":{"size":"1"}}}`)
	m.reload()
	m.relayout()
	if p, _ := m.paneOf("a.md"); p.rect.W != 80 {
		t.Fatalf("a 1/2 alone in its row is stretched to the whole width: %+v", p.rect)
	}
	right, _ := edges(t, m, "a.md")
	for _, dx := range []int{-1, -5} {
		drag(m, right, [2]int{right[0] + dx, right[1]})
		if got := viewsOf(t, dir)["a.md"].Size; got != "1/2" {
			t.Fatalf("a drag of %d cells wrote %q", dx, got)
		}
	}
	// Two thirds of the way back is past the middle between 1/2 and 1/3.
	drag(m, right, [2]int{59, right[1]})
	if got := viewsOf(t, dir)["a.md"].Size; got != "1/3" {
		t.Errorf("a drag to 60 of 80 stretched cells is 30 of 40: want 1/3, got %q", got)
	}
}

func TestZeroMakesALogAsTallAsItsContent(t *testing.T) {
	t.Parallel()
	m, dir := newModel(t, map[string]string{"build.log": "one\ntwo\n", "z.md": opened("other\n")})
	writeView(t, dir, `{"notes":{"build.log":{"open":true,"size":"1"},"z.md":{"size":"1"}}}`)
	m.reload()
	m.setFocus("build.log")
	m.relayout()
	if got := m.rowsOf(m.items[m.index("build.log")]); got != 10 {
		t.Fatalf("a log is ten lines tall by its kind: %d", got)
	}
	press(m, "0")
	if got := viewsOf(t, dir)["build.log"].Rows; got != store.ContentRows {
		t.Fatalf("0 on a log should write that it is as tall as its content: rows = %d", got)
	}
	if got := m.rowsOf(m.items[m.index("build.log")]); got != 0 {
		t.Errorf("the content height wins over the kind's: %d", got)
	}
	press(m, "shift+up")
	p, _ := m.paneOf("build.log")
	if got := viewsOf(t, dir)["build.log"].Rows; got != p.total+1 {
		t.Errorf("shift+up grows from the content (%d lines): rows = %d", p.total, got)
	}
}

func TestADoubleClickOnALogsBottomBorderGoesBackToTheContent(t *testing.T) {
	t.Parallel()
	m, dir := newModel(t, map[string]string{"build.log": "one\ntwo\n", "z.md": opened("other\n")})
	writeView(t, dir, `{"notes":{"build.log":{"open":true,"size":"1"},"z.md":{"size":"1"}}}`)
	m.reload()
	m.relayout()
	_, bottom := edges(t, m, "build.log")
	click(m, bottom[0], bottom[1])
	click(m, bottom[0], bottom[1])
	if got := viewsOf(t, dir)["build.log"].Rows; got != store.ContentRows {
		t.Errorf("rows = %d, want the content height", got)
	}
}

func TestANoteWithAHeightGetsItBeforeTheOthersShareTheRest(t *testing.T) {
	t.Parallel()
	m, dir := newModel(t, map[string]string{"a.md": opened("one\n"), "b.md": opened("two\n")})
	writeView(t, dir, `{"notes":{"a.md":{"size":"1","rows":5},"b.md":{"size":"1"}}}`)
	m.reload()
	m.relayout()
	if p, _ := m.paneOf("a.md"); p.rect.H != 7 {
		t.Fatalf("five rows and the borders are 7 lines, got %+v", p.rect)
	}
	press(m, "shift+down")
	if p, _ := m.paneOf("a.md"); p.rect.H != 6 {
		t.Errorf("shift+down should show: %+v", p.rect)
	}
	if b, _ := m.paneOf("b.md"); b.rect.Y+b.rect.H != m.bodyHeight() {
		t.Errorf("the other note takes the rest of the screen: %+v", b.rect)
	}
}
