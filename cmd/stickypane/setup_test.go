package main

import (
	"errors"
	"strings"
	"testing"
)

// setup prints its plan and runs nothing until told to: --yes runs it, and
// --dry-run only prints it. The scope, MCP and agents are flags.
func TestSetupAsksBeforeItRuns(t *testing.T) {
	project(t)
	var ran []string
	oldRun, oldCheck, oldLook, oldHas := setupRun, setupCheck, setupLook, setupHas
	t.Cleanup(func() { setupRun, setupCheck, setupLook, setupHas = oldRun, oldCheck, oldLook, oldHas })
	setupRun = func(cmd []string) error { ran = append(ran, strings.Join(cmd, " ")); return nil }
	setupCheck = func([]string) error { return errors.New("not registered") }
	setupLook = func(name string) (string, error) {
		if name == "claude" {
			return "/bin/claude", nil
		}
		return "", errors.New("missing")
	}
	setupHas = func(string) func() (bool, bool) { return func() (bool, bool) { return false, false } }

	code, out, _ := exec(t, "setup", "--scope", "project", "--mcp")
	if code != 0 || len(ran) != 0 || !strings.Contains(out, "claude plugin install board@stickypane --scope project") || !strings.Contains(out, "--yes") {
		t.Fatalf("without --yes nothing runs: code %d, ran %q\n%s", code, ran, out)
	}
	if code, _, _ := exec(t, "setup", "--dry-run", "--yes"); code != 0 || len(ran) != 0 {
		t.Errorf("--dry-run runs nothing: %q", ran)
	}
	code, out, errOut := exec(t, "setup", "--scope", "project", "--mcp", "--yes")
	if code != 0 || len(ran) != 3 || ran[2] != "claude mcp add --scope project stickypane -- stickypane mcp" {
		t.Fatalf("--yes runs the plan: code %d, ran %q, out %q, err %q", code, ran, out, errOut)
	}
	if code, _, errOut := exec(t, "setup", "--scope", "everywhere"); code != exitUsage || !strings.Contains(errOut, "user, project or local") {
		t.Errorf("a bad scope is a usage error: %d %q", code, errOut)
	}
	if code, _, _ := exec(t, "setup", "--agents", "gemini"); code != exitUsage {
		t.Errorf("an unknown agent is a usage error: %d", code)
	}
}
