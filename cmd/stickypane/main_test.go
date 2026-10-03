package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/LeeSwallow/stickypane/internal/initcmd"
)

func exec(t *testing.T, args ...string) (code int, stdout, stderr string) {
	t.Helper()
	return execIn(t, "", args...)
}

// execIn runs the command line with the given standard input.
func execIn(t *testing.T, stdin string, args ...string) (code int, stdout, stderr string) {
	t.Helper()
	var out, errOut bytes.Buffer
	code = run(args, strings.NewReader(stdin), &out, &errOut)
	return code, out.String(), errOut.String()
}

func TestGuidePrintsTheAgentGuide(t *testing.T) {
	code, out, _ := exec(t, "guide")
	if code != 0 || !strings.HasPrefix(out, initcmd.GuideText()) || !strings.Contains(out, "## Skills") {
		t.Errorf("code = %d, out = %q", code, out)
	}
	if code, out, _ = exec(t, "guide", "tracking-progress"); code != 0 || !strings.HasPrefix(out, "# Tracking progress") {
		t.Errorf("a skill: code = %d, out = %.80q", code, out)
	}
	if code, _, _ = exec(t, "guide", "nope"); code != 2 {
		t.Errorf("an unknown topic: code = %d", code)
	}
}

func TestVersionAndHelp(t *testing.T) {
	for _, arg := range []string{"version", "--version", "-v"} {
		if code, out, _ := exec(t, arg); code != 0 || out != "stickypane dev\n" {
			t.Errorf("%s: code = %d, out = %q", arg, code, out)
		}
	}
	for _, arg := range []string{"help", "--help", "-h"} {
		if code, out, _ := exec(t, arg); code != 0 || !strings.Contains(out, "stickypane init") {
			t.Errorf("%s: code = %d, out = %q", arg, code, out)
		}
	}
}

func TestUsageErrors(t *testing.T) {
	for _, args := range [][]string{{"--nope"}, {"a", "b"}, {"init", "--nope"}} {
		if code, _, errOut := exec(t, args...); code != 2 || errOut == "" {
			t.Errorf("%v: code = %d, stderr = %q; want 2 and a message", args, code, errOut)
		}
	}
}

func TestInitSetsUpTheCurrentDirectory(t *testing.T) {
	root := t.TempDir()
	t.Chdir(root)
	code, out, errOut := exec(t, "init", "--no-agent-docs")
	if code != 0 {
		t.Fatalf("code = %d, stderr = %q", code, errOut)
	}
	if _, err := os.Stat(filepath.Join(root, ".sticky", "welcome.md")); err != nil {
		t.Errorf("welcome note is missing: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "AGENTS.md")); err == nil {
		t.Error("--no-agent-docs must not create AGENTS.md")
	}
	if !strings.Contains(out, ".sticky") {
		t.Errorf("out = %q", out)
	}
}

func TestBoardWithoutNotesFolderExplainsInit(t *testing.T) {
	code, _, errOut := exec(t, t.TempDir())
	if code != 1 || !strings.Contains(errOut, "stickypane init") {
		t.Errorf("code = %d, stderr = %q", code, errOut)
	}
}
