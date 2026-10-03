package app

import (
	"errors"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
)

var errTest = errors.New("cannot read")

// typeKeys sends each character as its own key, and "<esc>" and "<enter>"
// as those keys.
func typeKeys(m *Model, script string) {
	for script != "" {
		switch {
		case strings.HasPrefix(script, "<esc>"):
			press(m, "esc")
			script = script[len("<esc>"):]
		case strings.HasPrefix(script, "<enter>"):
			press(m, "enter")
			script = script[len("<enter>"):]
		default:
			r := []rune(script)[0]
			if r == ' ' {
				press(m, "space")
			} else {
				m.Update(tea.KeyPressMsg{Code: r, Text: string(r)})
			}
			script = script[len(string(r)):]
		}
	}
}

func TestEOpensTheNoteInTheEditor(t *testing.T) {
	m, dir := newModel(t, map[string]string{"a.md": opened("first line\nsecond line\n")})
	press(m, "e")
	if m.mode != modeEdit {
		t.Fatalf("mode = %v", m.mode)
	}
	s := screen(m)
	for _, want := range []string{"a.md", "1 ---", "2 open: true", "4 first line", "5 second line"} {
		if !strings.Contains(s, want) {
			t.Errorf("the editor should show the file with line numbers, missing %q:\n%s", want, s)
		}
	}
	typeKeys(m, "GA, changed")
	if s := screen(m); !strings.Contains(s, "-- INSERT --") || !strings.Contains(s, "a.md [+]") {
		t.Errorf("the screen should show the mode and that there are changes:\n%s", s)
	}
	typeKeys(m, "<esc>:wq<enter>")
	if got := readFile(t, dir, "a.md"); got != opened("first line\nsecond line, changed\n") {
		t.Errorf("file = %q", got)
	}
	if m.mode != modeBoard || !strings.Contains(screen(m), "second line, changed") {
		t.Errorf(":wq should save and go back to the board (mode %v):\n%s", m.mode, screen(m))
	}
}

func TestTheEditorDoesNotLoseChangesByAccident(t *testing.T) {
	m, dir := newModel(t, map[string]string{"a.md": opened("text\n")})
	press(m, "e")
	typeKeys(m, "Gx:q<enter>")
	if m.mode != modeEdit || !strings.Contains(screen(m), "q!") {
		t.Fatalf(":q with changes should stay and say how to leave:\n%s", screen(m))
	}
	if _, cmd := m.Update(key("ctrl+c")); cmd != nil || m.mode != modeEdit {
		t.Fatalf("ctrl+c must not quit the board while a note is being edited")
	}
	typeKeys(m, ":q!<enter>")
	if m.mode != modeBoard || readFile(t, dir, "a.md") != opened("text\n") {
		t.Errorf(":q! should leave without writing: mode %v, file %q", m.mode, readFile(t, dir, "a.md"))
	}
}

func TestSavingOverAFileThatChangedAsksFirst(t *testing.T) {
	m, dir := newModel(t, map[string]string{"a.md": opened("mine\n")})
	press(m, "e")
	typeKeys(m, "GA!<esc>")
	writeFile(t, dir, "a.md", opened("the agent wrote this\n"))
	typeKeys(m, ":w<enter>")
	if got := readFile(t, dir, "a.md"); got != opened("the agent wrote this\n") {
		t.Fatalf("a save over someone else's change must not go through: %q", got)
	}
	if s := screen(m); m.mode != modeEdit || !strings.Contains(s, "w!") || !strings.Contains(s, "e!") {
		t.Fatalf("the editor should say what to do:\n%s", s)
	}
	typeKeys(m, ":e!<enter>")
	if s := screen(m); !strings.Contains(s, "the agent wrote this") || strings.Contains(s, "[+]") {
		t.Fatalf(":e! should load the file again:\n%s", s)
	}
	typeKeys(m, "GA?<esc>")
	writeFile(t, dir, "a.md", opened("changed again\n"))
	typeKeys(m, ":w!<enter>")
	if got := readFile(t, dir, "a.md"); got != opened("the agent wrote this?\n") {
		t.Errorf(":w! should overwrite: %q", got)
	}
	if m.mode != modeEdit || strings.Contains(screen(m), "[+]") {
		t.Errorf(":w! saves and stays in the editor:\n%s", screen(m))
	}
}

func TestTheEditorReturnsToWhereItWasOpened(t *testing.T) {
	m, _ := newModel(t, map[string]string{"a.md": opened("text\n")})
	press(m, "z", "e")
	if m.mode != modeEdit {
		t.Fatalf("e in a zoomed note should edit it, mode = %v", m.mode)
	}
	typeKeys(m, ":q<enter>")
	if m.mode != modeZoom {
		t.Errorf("mode after :q = %v, want the zoomed note", m.mode)
	}
}

func TestANoteThatCannotBeReadCannotBeEdited(t *testing.T) {
	m, _ := newModel(t, map[string]string{"a.md": opened("text\n")})
	m.items[0].note.Err = errTest
	press(m, "e")
	if m.mode == modeEdit {
		t.Error("an unreadable note should not open in the editor")
	}
}

func TestCapitalEOpensTheExternalEditor(t *testing.T) {
	m, _ := newModel(t, map[string]string{"a.md": opened("text\n")})
	if _, cmd := m.Update(key("E")); cmd == nil || m.mode != modeBoard {
		t.Errorf("E should hand the note to $EDITOR")
	}
}

func TestTheWheelMovesInTheEditor(t *testing.T) {
	m, _ := newModel(t, map[string]string{"a.md": opened(numbered(60))})
	press(m, "e")
	wheel(m, 5, 5, true)
	wheel(m, 5, 5, true)
	if s := screen(m); !strings.Contains(s, "7:1") {
		t.Errorf("the wheel should move down through the file:\n%s", s)
	}
}

func TestTheEditorFollowsAFileItHasNotChanged(t *testing.T) {
	m, dir := newModel(t, map[string]string{"a.md": opened("first\n")})
	press(m, "e")
	writeFile(t, dir, "a.md", opened("the agent rewrote this\n"))
	m.Update(changedMsg{})
	if s := screen(m); !strings.Contains(s, "the agent rewrote this") || strings.Contains(s, "[+]") {
		t.Fatalf("an untouched buffer should follow the file:\n%s", s)
	}
	typeKeys(m, "GA mine<esc>")
	writeFile(t, dir, "a.md", opened("rewritten again\n"))
	m.Update(changedMsg{})
	s := screen(m)
	if !strings.Contains(s, "the agent rewrote this mine") || strings.Contains(s, "rewritten again") {
		t.Fatalf("a buffer with changes is never replaced behind the user's back:\n%s", s)
	}
	if !strings.Contains(s, "changed on disk") {
		t.Errorf("the user should be told the file changed:\n%s", s)
	}
}
