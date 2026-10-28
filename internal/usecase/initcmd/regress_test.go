package initcmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunKeepsTextAfterAStrayStartMarker(t *testing.T) {
	root := t.TempDir()
	agents := filepath.Join(root, "AGENTS.md")
	if err := os.WriteFile(agents, []byte("intro\n"+startMark+"\nuser text that matters\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	run(t, root, Options{})
	first := read(t, agents)
	run(t, root, Options{})
	got := read(t, agents)
	if !strings.Contains(got, "user text that matters") {
		t.Errorf("user text was deleted:\n%s", got)
	}
	if got != first || strings.Count(got, "## stickypane notes") != 1 {
		t.Errorf("a second run should change nothing:\n%s", got)
	}
}

func TestRunDoesNotStackGuidesWhenAnEndMarkerComesFirst(t *testing.T) {
	root := t.TempDir()
	agents := filepath.Join(root, "AGENTS.md")
	if err := os.WriteFile(agents, []byte(endMark+"\nnotes\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	run(t, root, Options{})
	run(t, root, Options{})
	got := read(t, agents)
	if n := strings.Count(got, "## stickypane notes"); n != 1 {
		t.Errorf("the guide appears %d times, want 1:\n%s", n, got)
	}
	if !strings.Contains(got, "notes\n") {
		t.Errorf("user text was deleted:\n%s", got)
	}
}
