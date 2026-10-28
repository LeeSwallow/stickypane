package registry

import (
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

var update = flag.Bool("update", false, "rewrite the golden files in testdata/home")

// golden is the home folder the fixtures show: what stickypane writes, and
// what the desktop app's tests read (internal/adapter/registry/testdata/home from
// the repository root).
const golden = "testdata/home"

// fixed is the clock of the golden files.
var fixed = time.Date(2026, 10, 28, 14, 2, 11, 0, time.FixedZone("KST", 9*60*60))

func fixedWriter(home string, pid int) Writer {
	return Writer{Home: home, PID: pid, Now: func() time.Time { return fixed }}
}

// writeGolden writes, with a fixed clock, a home where one board was set
// up, opened in the terminal (pid 4242, still running) and changed by a
// command, and served to the app by a bridge (pid 4343) that has ended.
func writeGolden(home string) {
	tui := fixedWriter(home, 4242)
	tui.Append(Event{Type: "project.init", Project: "/work/demo"})
	tui.Live("/work/demo", "tui")
	tui.Touch("/work/demo", "card")
	bridge := fixedWriter(home, 4343)
	bridge.Live("/work/demo", "bridge")()
}

// files lists the records under home, by path relative to it, leaving out
// the touch markers, which are empty and named by a hash.
func files(t *testing.T, home string) map[string]string {
	t.Helper()
	out := map[string]string{}
	err := filepath.WalkDir(home, func(p string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() || strings.HasPrefix(d.Name(), "touch-") {
			return err
		}
		b, err := os.ReadFile(p)
		rel, _ := filepath.Rel(home, p)
		out[filepath.ToSlash(rel)] = string(b)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	return out
}

func TestTheWriterWritesTheGoldenFiles(t *testing.T) {
	home := t.TempDir()
	writeGolden(home)
	got := files(t, home)
	if *update {
		os.RemoveAll(golden)
		for rel, body := range got {
			p := filepath.Join(golden, filepath.FromSlash(rel))
			if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
				t.Fatal(err)
			}
		}
	}
	want := files(t, golden)
	if len(got) != len(want) {
		t.Fatalf("files = %v, want %v", keys(got), keys(want))
	}
	for rel, body := range want {
		if got[rel] != body {
			t.Errorf("%s =\n%s\nwant\n%s", rel, got[rel], body)
		}
	}
}

func keys(m map[string]string) []string {
	var out []string
	for k := range m {
		out = append(out, k)
	}
	return out
}

// TestTheRunFileKeepsItsFormat pins the run file byte for byte as it was
// written before it was typed: the keys of a map, in sorted order.
func TestTheRunFileKeepsItsFormat(t *testing.T) {
	b, err := os.ReadFile(filepath.Join(golden, "run", "4242.json"))
	if err != nil {
		t.Fatal(err)
	}
	want := `{"kind":"tui","pid":4242,"project":"/work/demo","started":"2026-10-28T14:02:11+09:00"}`
	if string(b) != want {
		t.Fatalf("run file = %s, want %s", b, want)
	}
}

func TestTouchIsThrottledByTheWritersClock(t *testing.T) {
	home := t.TempDir()
	now := fixed
	w := Writer{Home: home, PID: 1, Now: func() time.Time { return now }}
	log := filepath.Join(home, "events", "2026-10-28.jsonl")
	w.Touch("/p", "card")
	now = now.Add(59 * time.Second)
	w.Touch("/p", "card")
	if n := len(lines(t, log)); n != 1 {
		t.Fatalf("within the minute: lines = %d, want 1", n)
	}
	now = now.Add(2 * time.Second)
	w.Touch("/p", "card")
	if n := len(lines(t, log)); n != 2 {
		t.Fatalf("after the minute: lines = %d, want 2", n)
	}
}
