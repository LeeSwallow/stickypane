package app

import (
	"fmt"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
)

// bodyLines returns the screen without the title bar and the bottom line.
func bodyLines(m *Model) []string {
	lines := strings.Split(screen(m), "\n")
	return lines[len(m.bar) : len(lines)-1]
}

func TestOpenNotesFillTheScreen(t *testing.T) {
	m, _ := newModel(t, map[string]string{
		"a.md": "---\nopen: true\nsize: page\n---\nfirst\n",
		"b.md": "---\nopen: true\nsize: page\n---\nsecond\n",
	})
	body := bodyLines(m)
	if len(body) != 24-1-len(m.bar) {
		t.Fatalf("body = %d lines", len(body))
	}
	for i, l := range body {
		if w := len([]rune(strings.TrimRight(l, " "))); w != 80 {
			t.Errorf("line %d is %d cells wide, want the full 80: %q", i, w, l)
		}
	}
	if first, last := body[0], body[len(body)-1]; !strings.HasPrefix(first, "╔") || !strings.HasPrefix(last, "╰") {
		t.Errorf("the notes should reach from the top of the body to its bottom:\n%s", screen(m))
	}
}

func TestANoteAloneInItsRowTakesTheWholeWidth(t *testing.T) {
	m, _ := newModel(t, map[string]string{"a.md": "---\nopen: true\nsize: card\n---\nsmall\n"})
	if body := bodyLines(m); !strings.HasSuffix(body[0], "╗") || len([]rune(body[0])) != 80 {
		t.Errorf("no part of the screen should be empty:\n%s", screen(m))
	}
}

func TestALongNoteScrollsInsideItsPane(t *testing.T) {
	m, _ := newModel(t, map[string]string{
		"a.md": "---\nopen: true\nsize: page\n---\n" + numbered(60),
		"b.md": "---\nopen: true\nsize: page\n---\nthe other note\n",
	})
	s := screen(m)
	if !strings.Contains(s, "line 01") || !strings.Contains(s, "the other note") || strings.Contains(s, "line 30") {
		t.Fatalf("a long note is cut to its pane and the next note stays on the screen:\n%s", s)
	}
	if !strings.Contains(s, "█") {
		t.Errorf("a pane with more to show has a scroll bar:\n%s", s)
	}
	press(m, "j", "j")
	if s := screen(m); strings.Contains(s, "line 02") || !strings.Contains(s, "line 03") || !strings.Contains(s, "the other note") {
		t.Errorf("j should scroll inside the note and leave the rest alone:\n%s", s)
	}
	press(m, "G")
	if s := screen(m); !strings.Contains(s, "line 60") || !strings.Contains(s, "the other note") {
		t.Errorf("G should show the end of the note:\n%s", s)
	}
	press(m, "g")
	m.Update(tea.KeyPressMsg{Code: tea.KeyPgDown})
	if s := screen(m); strings.Contains(s, "line 01") || !strings.Contains(s, "the other note") {
		t.Errorf("page down should move by what the pane shows:\n%s", s)
	}
}

func TestTheWheelScrollsTheNoteUnderThePointer(t *testing.T) {
	m, _ := newModel(t, map[string]string{
		"a.md": "---\nopen: true\nsize: page\n---\nfocused note\n",
		"b.md": "---\nopen: true\nsize: page\n---\n" + numbered(60),
	})
	x, y := find(t, m, "line 02")
	wheel(m, x, y, true)
	if s := screen(m); strings.Contains(s, "line 03") || !strings.Contains(s, "line 04") || m.focus != "a.md" {
		t.Fatalf("the wheel should scroll the note it is over, whichever has the focus:\n%s", s)
	}
	wheel(m, x, y, false)
	wheel(m, x, y, false)
	if s := screen(m); !strings.Contains(s, "line 01") {
		t.Errorf("wheel up should scroll back to the top:\n%s", s)
	}
	fx, fy := find(t, m, "focused note")
	wheel(m, fx, fy, true)
	if s := screen(m); !strings.Contains(s, "focused note") {
		t.Errorf("a note that fits does not scroll:\n%s", s)
	}
}

func TestNotesThatDoNotFitGoToTheNextScreen(t *testing.T) {
	files := map[string]string{}
	for i := 1; i <= 5; i++ {
		files[fmt.Sprintf("n%d.md", i)] = fmt.Sprintf("---\nopen: true\nsize: page\n---\nnote %d\n%s", i, numbered(30))
	}
	m, _ := newModel(t, files)
	s := screen(m)
	if !strings.Contains(s, "note 1") || !strings.Contains(s, "note 2") || strings.Contains(s, "note 3") {
		t.Fatalf("two notes fit a screen of 24 lines; the third waits on the next:\n%s", s)
	}
	if !strings.HasSuffix(strings.Split(s, "\n")[0], " 1/3") {
		t.Errorf("the screen should say which screen it is:\n%s", s)
	}
	press(m, "]")
	if s := screen(m); !strings.Contains(s, "note 3") || !strings.Contains(s, " 2/3") || m.focus != "n3.md" {
		t.Fatalf("] should show the next screen and focus its first note (focus %q):\n%s", m.focus, s)
	}
	press(m, "]", "]", "]")
	if s := screen(m); !strings.Contains(s, "note 5") || !strings.Contains(s, " 3/3") {
		t.Errorf("] stops at the last screen:\n%s", s)
	}
	press(m, "[", "[", "[")
	if s := screen(m); !strings.Contains(s, "note 1") || !strings.Contains(s, " 1/3") {
		t.Errorf("[ should go back to the first screen:\n%s", s)
	}
	press(m, "tab", "tab")
	if s := screen(m); m.focus != "n3.md" || !strings.Contains(s, "note 3") {
		t.Errorf("moving the focus to a note on another screen shows that screen:\n%s", s)
	}
	if s := screen(m); !strings.Contains(s, "[ ] screen") {
		t.Errorf("the bottom line should offer the screen keys:\n%s", s)
	}
}

func TestTheCursorStaysInViewInsideAPane(t *testing.T) {
	var items strings.Builder
	for i := 1; i <= 40; i++ {
		fmt.Fprintf(&items, "- [ ] item %02d\n", i)
	}
	m, _ := newModel(t, map[string]string{
		"c.md": "---\ntype: checklist\nopen: true\n---\n" + items.String(),
		"z.md": "---\nopen: true\nsize: page\n---\n" + numbered(40),
	})
	for i := 0; i < 39; i++ {
		press(m, "j")
		if s := screen(m); !strings.Contains(s, "› ☐ item") {
			t.Fatalf("the cursor left the pane after %d steps:\n%s", i+1, s)
		}
	}
	if s := screen(m); !strings.Contains(s, "› ☐ item 40") || !strings.Contains(s, "line 01") {
		t.Errorf("the checklist should have scrolled to its last item, the other note untouched:\n%s", s)
	}
}

func TestALogFollowsItsEndUntilItIsScrolled(t *testing.T) {
	m, dir := newModel(t, map[string]string{
		"l.md": "---\ntype: log\nopen: true\n---\n" + numbered(40),
		"z.md": "---\nopen: true\nsize: page\n---\n" + numbered(40),
	})
	if s := screen(m); !strings.Contains(s, "line 40") {
		t.Fatalf("a log shows its end:\n%s", s)
	}
	writeFile(t, dir, "l.md", "---\ntype: log\nopen: true\n---\n"+numbered(40)+"the newest entry\n")
	press(m, "r")
	if s := screen(m); !strings.Contains(s, "the newest entry") {
		t.Fatalf("a log follows the file as it grows:\n%s", s)
	}
	press(m, "k", "k", "k")
	writeFile(t, dir, "l.md", "---\ntype: log\nopen: true\n---\n"+numbered(40)+"the newest entry\nand one more\n")
	press(m, "r")
	if s := screen(m); strings.Contains(s, "and one more") {
		t.Errorf("a log the user scrolled up stays where it was put:\n%s", s)
	}
	press(m, "G")
	writeFile(t, dir, "l.md", "---\ntype: log\nopen: true\n---\n"+numbered(40)+"the newest entry\nand one more\nand the last\n")
	press(m, "r")
	if s := screen(m); !strings.Contains(s, "and the last") {
		t.Errorf("scrolled back to its end, the log follows again:\n%s", s)
	}
}

func TestTinyScreensStillDraw(t *testing.T) {
	m, _ := newModel(t, map[string]string{"a.md": opened(numbered(30)), "c.md": checklistFile})
	for _, size := range [][2]int{{1, 1}, {8, 3}, {20, 5}, {40, 9}, {200, 60}} {
		m.Update(tea.WindowSizeMsg{Width: size[0], Height: size[1]})
		lines := strings.Split(m.render(), "\n")
		if len(lines) > size[1] {
			t.Errorf("%v: %d lines on the screen", size, len(lines))
		}
		press(m, "j", "]", "[", "tab")
	}
}
