package main

import (
	"bytes"
	"context"
	"strings"
	"testing"
)

// Flags may come before, between or after the words, and a negative number
// is a value, not a flag.
func TestFlagsAnywhereAndNegativeNumbers(t *testing.T) {
	project(t)
	if code, out, errOut := exec(t, "card", "work", "add", "--to", "Doing", "login API"); code != 0 || !strings.Contains(out, "work.md") {
		t.Fatalf("--to before the card: %d %q %q", code, out, errOut)
	}
	if code, out, errOut := exec(t, "chart", "tokens", "add", "delta", "-5"); code != 0 || !strings.Contains(out, "-5") {
		t.Errorf("a negative value: %d %q %q", code, out, errOut)
	}
	if code, out, _ := exec(t, "log", "worklog", "--", "--time", "is", "text", "here"); code != 0 || !strings.Contains(out, "worklog.md") {
		t.Errorf("after -- everything is words: %d %q", code, out)
	}
	if b, _ := exec2(t, "cat", "worklog"); !strings.Contains(b, "--time is text here") {
		t.Errorf("the log line = %q", b)
	}
}

// A flag the command does not have is a usage error, and the usage printed
// is that command's, not the whole page.
func TestUsageErrorsNameTheCommand(t *testing.T) {
	project(t)
	code, out, errOut := exec(t, "todo", "plan", "add", "x", "--time")
	if code != exitUsage || out != "" {
		t.Fatalf("--time on todo: code = %d, out = %q", code, out)
	}
	if !strings.Contains(errOut, "stickypane todo <name>") || strings.Contains(errOut, "stickypane mcp") {
		t.Errorf("only todo's usage:\n%s", errOut)
	}
}

// Interrupting wait ends it with the shells' code for Ctrl-C.
func TestAnInterruptedWaitExits130(t *testing.T) {
	project(t)
	if code, _, _ := execIn(t, "---\ntype: form\n---\n- ( ) a\n", "write", "ask"); code != 0 {
		t.Fatal("write failed")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	var out, errOut bytes.Buffer
	if code := runCtx(ctx, []string{"wait", "ask"}, strings.NewReader(""), &out, &errOut); code != exitInterrupted {
		t.Errorf("code = %d, stderr = %q", code, errOut.String())
	}
}

func exec2(t *testing.T, args ...string) (string, int) {
	t.Helper()
	code, out, _ := exec(t, args...)
	return out, code
}
