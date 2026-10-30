package app

import (
	"context"
	"fmt"
	"io"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/LeeSwallow/stickypane/internal/core/layout"
)

// rectsOf returns the panes of the notes, in the order named.
func rectsOf(t *testing.T, m *Model, names ...string) []layout.Rect {
	t.Helper()
	var out []layout.Rect
	for _, n := range names {
		p, ok := m.paneOf(n)
		if !ok {
			t.Fatalf("%s has no pane", n)
		}
		out = append(out, p.rect)
	}
	return out
}

func TestTwoNotesKeepTheirSizes(t *testing.T) {
	t.Parallel()
	m, dir := newModel(t, map[string]string{"a.md": opened(numbered(3)), "b.md": opened(numbered(20))})
	writeView(t, dir, `{"notes":{"a.md":{"size":"1/3","rows":6},"b.md":{"size":"1/2"}}}`)
	m.Update(tea.WindowSizeMsg{Width: 160, Height: 48})
	m.reload()
	m.relayout()
	// a third and a half of 160 cells; a is 6 lines and its borders, b
	// its 20 lines and its borders, and the rest of the row is empty.
	want := []layout.Rect{{X: 0, Y: 0, W: 53, H: 8}, {X: 53, Y: 0, W: 80, H: 22}}
	if got := rectsOf(t, m, "a.md", "b.md"); fmt.Sprint(got) != fmt.Sprint(want) {
		t.Errorf("panes = %v, want %v\n%s", got, want, screen(m))
	}
	// Shrinking a note makes it smaller, with the other note beside it.
	press(m, "shift+down")
	writeView(t, dir, `{"notes":{"a.md":{"size":"1/4","rows":5},"b.md":{"size":"1/2"}}}`)
	m.reload()
	m.relayout()
	want = []layout.Rect{{X: 0, Y: 0, W: 40, H: 7}, {X: 40, Y: 0, W: 80, H: 22}}
	if got := rectsOf(t, m, "a.md", "b.md"); fmt.Sprint(got) != fmt.Sprint(want) {
		t.Errorf("after shrinking: panes = %v, want %v", got, want)
	}
}

func TestManyNotesKeepTheirSizes(t *testing.T) {
	t.Parallel()
	files := map[string]string{}
	for _, n := range []string{"a", "b", "c", "d", "e"} {
		files[n+".md"] = opened(numbered(4))
	}
	m, dir := newModel(t, files)
	writeView(t, dir, `{"notes":{
		"a.md":{"size":"1/4","rows":3},"b.md":{"size":"1/4","rows":8},"c.md":{"size":"1/3"},
		"d.md":{"size":"1/2","rows":2},"e.md":{"size":"1/3"}}}`)
	m.Update(tea.WindowSizeMsg{Width: 160, Height: 48})
	m.reload()
	m.relayout()
	// The first row is 1/4 1/4 1/3, its notes at their own heights and
	// the next row under the tallest (b, 10 lines); d and e share the next.
	want := []layout.Rect{
		{X: 0, Y: 0, W: 40, H: 5}, {X: 40, Y: 0, W: 40, H: 10}, {X: 80, Y: 0, W: 53, H: 6},
		{X: 0, Y: 10, W: 80, H: 4}, {X: 80, Y: 10, W: 53, H: 6},
	}
	if got := rectsOf(t, m, "a.md", "b.md", "c.md", "d.md", "e.md"); fmt.Sprint(got) != fmt.Sprint(want) {
		t.Errorf("panes = %v, want %v\n%s", got, want, screen(m))
	}
}

func TestNotesThatDoNotFitAreSqueezedNotCut(t *testing.T) {
	t.Parallel()
	m, dir := newModel(t, map[string]string{"a.md": opened(numbered(40)), "b.md": opened(numbered(40))})
	writeView(t, dir, `{"notes":{"a.md":{"size":"1"},"b.md":{"size":"1","rows":30}}}`)
	m.reload()
	m.relayout()
	a, b := rectsOf(t, m, "a.md", "b.md")[0], rectsOf(t, m, "b.md")[0]
	if a.H+b.H != m.bodyHeight() || a.H < minPane || b.H < minPane || m.screens != 1 {
		t.Errorf("two long notes share the screen: %v %v of %d lines, %d screens", a, b, m.bodyHeight(), m.screens)
	}
}

// TestADragThroughTheTerminalResizesANote runs the program on the bytes a
// terminal sends for a press on a note's border, a move with the button
// held and a release (SGR mouse reports, as tmux and most terminals send
// them), and checks the note's size and height in sticky.json.
func TestADragThroughTheTerminalResizesANote(t *testing.T) {
	t.Parallel()
	m, dir := newModel(t, map[string]string{"a.md": opened("one\n"), "b.md": opened(numbered(12))})
	writeView(t, dir, `{"notes":{"a.md":{"size":"1/2","rows":8},"b.md":{"size":"1/2"}}}`)
	m.reload()
	m.relayout()
	right, _ := edges(t, m, "a.md")
	_, bottom := edges(t, m, "a.md")

	in, feed := io.Pipe()
	sized := make(chan struct{})
	var once bool
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	p := tea.NewProgram(m, tea.WithContext(ctx), tea.WithInput(in), tea.WithOutput(io.Discard),
		tea.WithWindowSize(80, 24), tea.WithoutSignals(),
		tea.WithFilter(func(_ tea.Model, msg tea.Msg) tea.Msg {
			if _, ok := msg.(tea.WindowSizeMsg); ok && !once {
				once = true
				close(sized)
			}
			return msg
		}))
	done := make(chan error, 1)
	go func() { _, err := p.Run(); done <- err }()
	<-sized
	// SGR reports count cells from 1.
	sgr := func(button, x, y int, end byte) string { return fmt.Sprintf("\x1b[<%d;%d;%d%c", button, x+1, y+1, end) }
	input := sgr(0, right[0], right[1], 'M') + // press the right border
		sgr(32, right[0]-6, right[1], 'M') + sgr(32, right[0]-12, right[1], 'M') + // move with it held
		sgr(0, right[0]-12, right[1], 'm') + // let go: 28 cells, a third
		sgr(0, bottom[0], bottom[1], 'M') + // the bottom border
		sgr(32, bottom[0], bottom[1]-2, 'M') + sgr(32, bottom[0], bottom[1]-3, 'M') +
		sgr(0, bottom[0], bottom[1]-3, 'm') + // three lines shorter
		"q"
	if _, err := io.WriteString(feed, input); err != nil {
		t.Fatal(err)
	}
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-ctx.Done():
		t.Fatal("the program did not quit")
	}
	feed.Close()
	v := viewsOf(t, dir)["a.md"]
	if v.Size != "1/3" || v.Rows != 5 {
		t.Errorf("a.md = size %q rows %d, want 1/3 and 5", v.Size, v.Rows)
	}
	if got := rectsOf(t, m, "a.md")[0]; got != (layout.Rect{X: 0, Y: 0, W: 26, H: 7}) {
		t.Errorf("a.md is drawn at %v, want a third of 80 and 5 lines", got)
	}
}
