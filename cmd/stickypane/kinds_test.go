package main

import (
	"strings"
	"testing"
)

// `stickypane kinds` lists every shape with what it is for and the fastest
// way to make one; it needs no board.
func TestKindsListsEveryShape(t *testing.T) {
	t.Chdir(t.TempDir())
	code, out, errOut := exec(t, "kinds")
	if code != 0 {
		t.Fatalf("code = %d, stderr = %q", code, errOut)
	}
	for _, want := range []string{"note", "board", "checklist", "log", "chart", "form", "script", "tab", "book", "stickypane card", "stickypane todo", "stickypane kinds <shape>"} {
		if !strings.Contains(out, want) {
			t.Errorf("kinds should mention %q:\n%s", want, out)
		}
	}
}

// `stickypane kinds board` prints one shape in full: the command, a file to
// copy, how it behaves and its keys.
func TestKindsPrintsOneShapeInFull(t *testing.T) {
	t.Chdir(t.TempDir())
	code, out, _ := exec(t, "kinds", "board")
	if code != 0 {
		t.Fatalf("code = %d", code)
	}
	for _, want := range []string{"stickypane card", "type: board", "## To do", "column", "H L"} {
		if !strings.Contains(out, want) {
			t.Errorf("kinds board should show %q:\n%s", want, out)
		}
	}
	if code, out, _ := exec(t, "kinds", "script"); code != 0 || !strings.Contains(out, ".sh") {
		t.Errorf("a script is a .sh file: %d\n%s", code, out)
	}
	if code, _, errOut := exec(t, "kinds", "teapot"); code != 2 || !strings.Contains(errOut, "checklist") {
		t.Errorf("an unknown shape lists the shapes: %d, %q", code, errOut)
	}
}
