package initcmd

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/LeeSwallow/stickypane/internal/doc"
)

func read(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func run(t *testing.T, root string, opts Options) string {
	t.Helper()
	var out bytes.Buffer
	if err := Run(root, opts, &out); err != nil {
		t.Fatal(err)
	}
	return out.String()
}

func TestGuideIsShortAndCoversEveryShape(t *testing.T) {
	g := Guide()
	if !strings.HasPrefix(g, startMark+"\n") || !strings.HasSuffix(g, endMark+"\n") {
		t.Error("the guide must be wrapped in the markers")
	}
	if n := strings.Count(g, "\n"); n > 40 {
		t.Errorf("the guide is %d lines, want at most 40", n)
	}
	for _, want := range []string{".stickypane/", "type: board", "type: checklist", "type: log", "type: chart", "label: number", "mermaid", "- [ ]", "## Heading", "color", "pin: true", "open: true", "`size`"} {
		if !strings.Contains(g, want) {
			t.Errorf("the guide should mention %q", want)
		}
	}
}

func TestRunCreatesBoardWelcomeNoteAndAgentsFile(t *testing.T) {
	root := t.TempDir()
	out := run(t, root, Options{})
	welcome := doc.Parse([]byte(read(t, filepath.Join(root, ".stickypane", "welcome.md"))))
	if open, _ := welcome.Get("open"); open != "true" {
		t.Errorf("the welcome note should be open on first run, open = %q", open)
	}
	if !welcome.Pinned() || !strings.Contains(welcome.Body, "jots a note") {
		t.Errorf("the welcome note should be pinned and explain the keys: %q", welcome.Body)
	}
	if got := read(t, filepath.Join(root, "AGENTS.md")); got != Guide() {
		t.Errorf("AGENTS.md = %q, want exactly the guide", got)
	}
	if _, err := os.Stat(filepath.Join(root, "CLAUDE.md")); !errors.Is(err, os.ErrNotExist) {
		t.Error("CLAUDE.md should not be created when it did not exist")
	}
	if !strings.Contains(out, "AGENTS.md") || !strings.Contains(out, "stickypane") {
		t.Errorf("output should say what happened: %q", out)
	}
}

func TestRunAppendsToExistingAgentFilesOnly(t *testing.T) {
	root := t.TempDir()
	claude := filepath.Join(root, "CLAUDE.md")
	if err := os.WriteFile(claude, []byte("# Project rules\n\nUse tabs."), 0o644); err != nil {
		t.Fatal(err)
	}
	run(t, root, Options{})
	if got, want := read(t, claude), "# Project rules\n\nUse tabs.\n\n"+Guide(); got != want {
		t.Errorf("CLAUDE.md = %q\nwant %q", got, want)
	}
	if _, err := os.Stat(filepath.Join(root, "AGENTS.md")); !errors.Is(err, os.ErrNotExist) {
		t.Error("AGENTS.md should not be created when CLAUDE.md exists")
	}
}

func TestRunTwiceChangesNothing(t *testing.T) {
	root := t.TempDir()
	run(t, root, Options{})
	agents := filepath.Join(root, "AGENTS.md")
	welcome := filepath.Join(root, ".stickypane", "welcome.md")
	first := read(t, agents)
	if err := os.Remove(welcome); err != nil { // the user took the note down
		t.Fatal(err)
	}
	run(t, root, Options{})
	if got := read(t, agents); got != first {
		t.Errorf("a second run changed AGENTS.md:\n%q\nwas:\n%q", got, first)
	}
	if _, err := os.Stat(welcome); !errors.Is(err, os.ErrNotExist) {
		t.Error("a second run must not bring the welcome note back")
	}
}

func TestRunReplacesAnOldGuide(t *testing.T) {
	root := t.TempDir()
	agents := filepath.Join(root, "AGENTS.md")
	old := "intro\n\n" + startMark + "\nstale text\n" + endMark + "\n\noutro\n"
	if err := os.WriteFile(agents, []byte(old), 0o644); err != nil {
		t.Fatal(err)
	}
	run(t, root, Options{})
	if got, want := read(t, agents), "intro\n\n"+Guide()+"\noutro\n"; got != want {
		t.Errorf("AGENTS.md = %q\nwant %q", got, want)
	}
}

func TestNoAgentDocs(t *testing.T) {
	root := t.TempDir()
	run(t, root, Options{NoAgentDocs: true})
	if _, err := os.Stat(filepath.Join(root, ".stickypane")); err != nil {
		t.Errorf("the board should still be created: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "AGENTS.md")); !errors.Is(err, os.ErrNotExist) {
		t.Error("AGENTS.md should not be touched")
	}
}

func TestGuideTextHasNoMarkers(t *testing.T) {
	text := GuideText()
	if strings.Contains(text, "stickypane:start") || strings.Contains(text, "stickypane:end") {
		t.Errorf("GuideText should drop the markers: %q", text)
	}
	if !strings.HasPrefix(text, "## stickypane notes") || !strings.Contains(text, "type: board") {
		t.Errorf("GuideText should be the guide itself: %q", text)
	}
}
