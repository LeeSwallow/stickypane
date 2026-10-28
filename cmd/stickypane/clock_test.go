package main

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/LeeSwallow/stickypane/internal/store"
)

// The command line reads one clock: log --time, a new note's created and a
// ticked item all say its time.
func TestTheCommandLineWritesTheTimeOfItsClock(t *testing.T) {
	root := project(t)
	at := time.Date(2026, 10, 3, 14, 2, 5, 0, time.Local)
	clocked := func(args ...string) {
		t.Helper()
		var out, errOut bytes.Buffer
		e := env{ctx: context.Background(), stdin: strings.NewReader(""), stdout: &out, stderr: &errOut, now: func() time.Time { return at }}
		if code := runEnv(e, args); code != 0 {
			t.Fatalf("%v: code %d, %s", args, code, errOut.String())
		}
	}
	file := func(name string) string {
		t.Helper()
		b, err := os.ReadFile(filepath.Join(root, store.DirName, name))
		if err != nil {
			t.Fatal(err)
		}
		return string(b)
	}
	clocked("log", "worklog", "--time", "tests", "passed")
	clocked("todo", "plan", "add", "ship")
	clocked("todo", "plan", "check", "ship")
	if got := file("worklog.md"); !strings.Contains(got, "14:02 tests passed\n") || !strings.Contains(got, "created: 2026-10-03 14:02\n") {
		t.Errorf("worklog:\n%s", got)
	}
	if got := file("plan.md"); !strings.Contains(got, "- [x] ship ✅ 2026-10-03 14:02\n") {
		t.Errorf("plan:\n%s", got)
	}
}
