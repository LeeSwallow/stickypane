package app

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/LeeSwallow/stickypane/internal/when"
)

func TestZoomGivesOneNoteTheWholeScreen(t *testing.T) {
	m, _ := newModel(t, map[string]string{"long.md": opened(numbered(15)), "b.md": boardFile})
	press(m, "tab", "enter") // long.md is second: b.md sorts first
	if m.mode != modeZoom {
		t.Fatalf("mode = %v, want zoom", m.mode)
	}
	s := screen(m)
	if strings.Contains(s, "▦ Auth") || strings.Contains(s, "payments") {
		t.Errorf("zoom should hide the title bar and the other notes:\n%s", s)
	}
	if !strings.Contains(s, "line 01") || !strings.Contains(s, "line 15") || !strings.Contains(s, "esc back") {
		t.Errorf("the zoomed note should show in full with its hint:\n%s", s)
	}
	if !strings.Contains(s, "long.md") {
		t.Errorf("an untitled zoomed note should be named by its file:\n%s", s)
	}
	press(m, "esc")
	if m.mode != modeBoard || !strings.Contains(screen(m), "payments") {
		t.Error("esc should return to the open notes")
	}
}

func TestZoomScrollsANoteWithoutACursor(t *testing.T) {
	m, _ := newModel(t, map[string]string{"long.md": opened(numbered(60))})
	press(m, "enter")
	if s := screen(m); !strings.Contains(s, "line 01") || strings.Contains(s, "line 60") {
		t.Fatalf("zoom should start at the top:\n%s", s)
	}
	press(m, "G")
	if s := screen(m); !strings.Contains(s, "line 60") || strings.Contains(s, "line 01") {
		t.Errorf("G should scroll to the end:\n%s", s)
	}
	press(m, "k", "k")
	if s := screen(m); strings.Contains(s, "line 60") {
		t.Errorf("k should scroll up:\n%s", s)
	}
}

func TestZoomReachesEveryLineOfANoteWithACursor(t *testing.T) {
	body := "---\ntype: checklist\ntitle: Mixed\nopen: true\n---\n" + numbered(60) + "- [ ] one\n- [ ] two\n"
	for i := 1; i <= 40; i++ {
		body += fmt.Sprintf("tail %02d\n", i)
	}
	m, _ := newModel(t, map[string]string{"c.md": body})
	press(m, "enter")
	if s := screen(m); !strings.Contains(s, "› ☐ one") {
		t.Fatalf("zoom should start where the cursor is:\n%s", s)
	}
	press(m, "g")
	if s := screen(m); !strings.Contains(s, "line 01") {
		t.Errorf("g should reach the top of a zoomed note even though it has a cursor:\n%s", s)
	}
	press(m, "G")
	if s := screen(m); !strings.Contains(s, "tail 40") {
		t.Errorf("G should reach the end:\n%s", s)
	}
	press(m, "j")
	if s := screen(m); !strings.Contains(s, "› ☐ two") {
		t.Errorf("a cursor key should bring the cursor back into view:\n%s", s)
	}
}

func TestZoomStartsAtTheEndOfALog(t *testing.T) {
	m, _ := newModel(t, map[string]string{"log.md": "---\ntype: log\nopen: true\n---\n" + numbered(60)})
	press(m, "enter")
	if s := screen(m); !strings.Contains(s, "line 60") || strings.Contains(s, "line 01") {
		t.Errorf("a zoomed log should show its end:\n%s", s)
	}
}

func TestZoomKeysReachTheWidgetAndFollowTheCursor(t *testing.T) {
	var sb strings.Builder
	sb.WriteString("---\ntype: checklist\ntitle: Long\nopen: true\n---\n")
	for i := 1; i <= 60; i++ {
		sb.WriteString("- [ ] step\n")
	}
	m, dir := newModel(t, map[string]string{"c.md": sb.String()})
	press(m, "enter")
	for i := 0; i < 40; i++ {
		press(m, "j")
	}
	if !strings.Contains(screen(m), "› ☐ step") {
		t.Fatalf("the cursor should stay visible while zoomed:\n%s", screen(m))
	}
	press(m, "space")
	lines := strings.Split(readFile(t, dir, "c.md"), "\n")
	if !strings.HasPrefix(lines[5+40], "- [x] step ✅ ") || strings.Count(strings.Join(lines, "\n"), "[x]") != 1 {
		t.Errorf("space should tick the 41st item only, got line %q", lines[5+40])
	}
}

func TestConflictIsReportedAndNothingIsWritten(t *testing.T) {
	m, dir := newModel(t, map[string]string{"b.md": boardFile})
	agent := strings.Replace(boardFile, "## To do\n- payments\n", "## To do\n", 1) + "- payments shipped\n"
	writeFile(t, dir, "b.md", agent) // the agent edits before the screen catches up
	press(m, "L")
	if got := plain(readFile(t, dir, "b.md")); got != agent {
		t.Errorf("the agent's version must survive, got %q", got)
	}
	s := screen(m)
	if !strings.Contains(s, "changed on disk") || !strings.Contains(s, "payments shipped") {
		t.Errorf("the conflict should be reported and the screen should catch up:\n%s", s)
	}
}

func TestFailedWriteRevertsTheScreen(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root can write everywhere")
	}
	m, dir := newModel(t, map[string]string{"c.md": checklistFile})
	if err := os.Chmod(dir, 0o555); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chmod(dir, 0o755) })
	press(m, "j", "space")
	if got := plain(readFile(t, dir, "c.md")); got != checklistFile {
		t.Fatalf("the write should have failed, file = %q", got)
	}
	s := screen(m)
	if !strings.Contains(s, "Write failed") {
		t.Errorf("the failure should be reported:\n%s", s)
	}
	if strings.Contains(s, "☑ ship") || !strings.Contains(s, "☐ ship") || !strings.Contains(s, "1/2") {
		t.Errorf("the screen must show the file as it is, not the change that failed:\n%s", s)
	}
}

func TestPromptAddsACard(t *testing.T) {
	m, dir := newModel(t, map[string]string{"b.md": boardFile})
	press(m, "n")
	if s := screen(m); !strings.Contains(s, "New card in To do:") {
		t.Fatalf("n on a focused board should ask for a card:\n%s", s)
	}
	typeText(m, "write 테스트")
	press(m, "enter")
	if got := plain(readFile(t, dir, "b.md")); !strings.Contains(got, "## To do\n- payments\n- write 테스트\n## Doing") {
		t.Errorf("file = %q", got)
	}
	if m.mode != modeBoard {
		t.Errorf("mode = %v, want the main screen", m.mode)
	}
}

func TestPromptCanBeCancelledOrLeftEmpty(t *testing.T) {
	m, dir := newModel(t, map[string]string{"b.md": boardFile})
	press(m, "enter", "n")
	typeText(m, "q") // "q" is text here, not a key
	press(m, "esc")
	press(m, "n")
	typeText(m, "   ")
	press(m, "enter")
	if got := plain(readFile(t, dir, "b.md")); got != boardFile {
		t.Errorf("nothing should have been written, got %q", got)
	}
	if m.mode != modeZoom {
		t.Errorf("mode = %v, want the zoom the prompt was opened from", m.mode)
	}
}

func TestZoomClosesWhenNoteIsRemoved(t *testing.T) {
	m, dir := newModel(t, map[string]string{"a.md": opened("one\n"), "b.md": boardFile})
	press(m, "tab", "enter")
	if err := os.Remove(filepath.Join(dir, "b.md")); err != nil {
		t.Fatal(err)
	}
	m.Update(changedMsg{})
	if m.mode != modeBoard {
		t.Fatalf("mode = %v, want the main screen", m.mode)
	}
	if s := screen(m); !strings.Contains(s, "The note was removed.") {
		t.Errorf("the user should be told:\n%s", s)
	}
	press(m, "enter", "j", "esc") // keeps working on the remaining note
}

func TestPromptClosesWhenNoteIsRemoved(t *testing.T) {
	m, dir := newModel(t, map[string]string{"a.md": opened("one\n"), "b.md": boardFile})
	press(m, "tab", "enter", "n")
	if err := os.Remove(filepath.Join(dir, "b.md")); err != nil {
		t.Fatal(err)
	}
	m.Update(changedMsg{})
	if m.mode != modeBoard {
		t.Fatalf("mode = %v, want the main screen", m.mode)
	}
	if s := screen(m); !strings.Contains(s, "The note was removed.") || !strings.Contains(s, "one") {
		t.Errorf("the main screen should be back with a message:\n%s", s)
	}
}

func TestZoomSurvivesKindChange(t *testing.T) {
	m, dir := newModel(t, map[string]string{"b.md": boardFile})
	press(m, "enter", "l")
	writeFile(t, dir, "b.md", checklistFile)
	m.Update(changedMsg{})
	if s := screen(m); !strings.Contains(s, "1/2") {
		t.Errorf("the zoomed note should now draw as a checklist:\n%s", s)
	}
	press(m, "space") // the open item is first on the screen
	if got := readFile(t, dir, "b.md"); !strings.Contains(got, "- [x] ship ✅ ") {
		t.Errorf("keys should reach the new widget, file = %q", got)
	}
}

func TestExternalEditKeepsTheCursor(t *testing.T) {
	m, dir := newModel(t, map[string]string{"b.md": boardFile})
	press(m, "l")
	writeFile(t, dir, "b.md", strings.Replace(boardFile, "- schema\n", "- schema\n- docs\n", 1))
	m.Update(changedMsg{})
	if s := screen(m); !strings.Contains(s, "› login API") || !strings.Contains(s, "docs") {
		t.Errorf("the cursor should stay and the new card should appear:\n%s", s)
	}
}

func TestZoomFitsAnySize(t *testing.T) {
	m, _ := newModel(t, map[string]string{"b.md": boardFile})
	press(m, "enter")
	for _, size := range [][2]int{{80, 24}, {40, 8}, {12, 4}, {3, 2}} {
		m.Update(tea.WindowSizeMsg{Width: size[0], Height: size[1]})
		if n := strings.Count(m.render(), "\n") + 1; n > size[1] {
			t.Errorf("%v: %d lines", size, n)
		}
	}
}

// Zoomed in, a note says when it was made and when it last changed.
func TestAZoomedNoteSaysWhenItWasMadeAndChanged(t *testing.T) {
	made := time.Now().Add(-48 * time.Hour).Truncate(time.Minute)
	m, dir := newModel(t, map[string]string{"plan.md": "---\nopen: true\ncreated: " + made.Format("2006-01-02 15:04") + "\n---\nthe plan\n"})
	changed := time.Now().Add(-time.Minute).Truncate(time.Minute)
	if err := os.Chtimes(filepath.Join(dir, "plan.md"), changed, changed); err != nil {
		t.Fatal(err)
	}
	m.Update(changedMsg{})
	press(m, "z")
	want := when.Short(made, time.Now()) + " → " + when.Short(changed, time.Now())
	if s := screen(m); !strings.Contains(s, want) {
		t.Errorf("want %q in the zoomed border:\n%s", want, s)
	}
}
