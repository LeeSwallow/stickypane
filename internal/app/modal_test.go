package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const checklistFile = "---\ntype: checklist\ntitle: Release\n---\n- [x] build\n- [ ] ship\n"

func TestEnterOpensTheWholeNote(t *testing.T) {
	var body strings.Builder
	for i := 1; i <= 15; i++ {
		body.WriteString("line ")
		body.WriteString(strings.Repeat("i", i))
		body.WriteString("\n")
	}
	m, _ := newModel(t, map[string]string{"long.md": body.String()})
	last := "line " + strings.Repeat("i", 15)
	if strings.Contains(screen(m), last) {
		t.Fatal("the board preview should not show the whole note")
	}
	press(m, "enter")
	s := screen(m)
	if !strings.Contains(s, last) || !strings.Contains(s, "esc close") {
		t.Errorf("the open note should show every line and the modal hint:\n%s", s)
	}
	if !strings.Contains(s, "long.md") {
		t.Errorf("an untitled open note should be named by its file:\n%s", s)
	}
	press(m, "esc")
	if strings.Contains(screen(m), last) || m.mode != modeBoard {
		t.Error("esc should return to the board")
	}
}

func TestMovingACardWritesTheFile(t *testing.T) {
	m, dir := newModel(t, map[string]string{"b.md": boardFile})
	press(m, "enter", "L")
	want := "---\ntype: board\ntitle: Auth\n---\n## To do\n## Doing\n- login API\n- payments\n## Done\n- schema\n"
	if got := readFile(t, dir, "b.md"); got != want {
		t.Errorf("file = %q\nwant  %q", got, want)
	}
	if s := screen(m); !strings.Contains(s, "› payments") {
		t.Errorf("the cursor should follow the card:\n%s", s)
	}
}

func TestConflictIsReportedAndNothingIsWritten(t *testing.T) {
	m, dir := newModel(t, map[string]string{"b.md": boardFile})
	press(m, "enter")
	agent := "---\ntype: board\ntitle: Auth\n---\n## To do\n## Doing\n- login API\n## Done\n- schema\n- payments shipped\n"
	writeFile(t, dir, "b.md", agent) // the agent edits before the screen catches up
	press(m, "L")
	if got := readFile(t, dir, "b.md"); got != agent {
		t.Errorf("the agent's version must survive, got %q", got)
	}
	s := screen(m)
	if !strings.Contains(s, "changed on disk") {
		t.Errorf("the conflict should be reported:\n%s", s)
	}
	if !strings.Contains(s, "payments shipped") {
		t.Errorf("the screen should catch up with the file:\n%s", s)
	}
}

func TestPromptAddsACard(t *testing.T) {
	m, dir := newModel(t, map[string]string{"b.md": boardFile})
	press(m, "enter", "n")
	if s := screen(m); !strings.Contains(s, "New card in To do:") {
		t.Fatalf("the prompt should show on the bottom line:\n%s", s)
	}
	typeText(m, "write 테스트")
	press(m, "enter")
	if got := readFile(t, dir, "b.md"); !strings.Contains(got, "## To do\n- payments\n- write 테스트\n## Doing") {
		t.Errorf("file = %q", got)
	}
	if m.mode != modeModal {
		t.Errorf("mode = %v, want the modal again", m.mode)
	}
}

func TestPromptCanBeCancelledOrLeftEmpty(t *testing.T) {
	m, dir := newModel(t, map[string]string{"b.md": boardFile})
	press(m, "enter", "n")
	typeText(m, "q") // "q" is text here, not the close key
	press(m, "esc")
	press(m, "n")
	typeText(m, "   ")
	press(m, "enter")
	if got := readFile(t, dir, "b.md"); got != boardFile {
		t.Errorf("nothing should have been written, got %q", got)
	}
	if m.mode != modeModal {
		t.Errorf("mode = %v, want the modal", m.mode)
	}
}

func TestSpaceTogglesAChecklistItem(t *testing.T) {
	m, dir := newModel(t, map[string]string{"c.md": checklistFile})
	press(m, "enter", "j", "space")
	if got := readFile(t, dir, "c.md"); !strings.HasSuffix(got, "- [x] build\n- [x] ship\n") {
		t.Errorf("file = %q", got)
	}
	press(m, "esc")
	if s := screen(m); !strings.Contains(s, "2/2") {
		t.Errorf("the board preview should show the new progress:\n%s", s)
	}
}

func TestModalClosesWhenNoteIsRemoved(t *testing.T) {
	m, dir := newModel(t, map[string]string{"a.md": "one\n", "b.md": boardFile})
	press(m, "tab", "enter")
	if err := os.Remove(filepath.Join(dir, "b.md")); err != nil {
		t.Fatal(err)
	}
	m.Update(changedMsg{})
	if m.mode != modeBoard {
		t.Fatalf("mode = %v, want the board", m.mode)
	}
	if s := screen(m); !strings.Contains(s, "The note was removed.") {
		t.Errorf("the user should be told:\n%s", s)
	}
	press(m, "enter", "j", "esc") // keeps working on the remaining note
}

func TestModalSurvivesKindChange(t *testing.T) {
	m, dir := newModel(t, map[string]string{"b.md": boardFile})
	press(m, "enter", "l")
	writeFile(t, dir, "b.md", checklistFile)
	m.Update(changedMsg{})
	if s := screen(m); !strings.Contains(s, "1/2") {
		t.Errorf("the open note should now draw as a checklist:\n%s", s)
	}
	press(m, "j", "space")
	if got := readFile(t, dir, "b.md"); !strings.HasSuffix(got, "- [x] ship\n") {
		t.Errorf("keys should reach the new widget, file = %q", got)
	}
}

func TestExternalEditKeepsModalCursor(t *testing.T) {
	m, dir := newModel(t, map[string]string{"b.md": boardFile})
	press(m, "enter", "l")
	writeFile(t, dir, "b.md", strings.Replace(boardFile, "- schema\n", "- schema\n- docs\n", 1))
	m.Update(changedMsg{})
	if s := screen(m); !strings.Contains(s, "› login API") || !strings.Contains(s, "docs") {
		t.Errorf("the cursor should stay and the new card should appear:\n%s", s)
	}
}
