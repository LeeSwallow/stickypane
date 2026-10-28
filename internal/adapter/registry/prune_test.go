package registry

import (
	"os"
	"path/filepath"
	"sort"
	"testing"
	"time"
)

// fakeAlive makes only the given pids alive for the length of the test.
func fakeAlive(t *testing.T, pids ...int) {
	t.Helper()
	old := alive
	t.Cleanup(func() { alive = old })
	alive = func(pid int) bool {
		for _, p := range pids {
			if p == pid {
				return true
			}
		}
		return false
	}
}

func put(t *testing.T, home, rel string, mod time.Time) {
	t.Helper()
	p := filepath.Join(home, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(p, mod, mod); err != nil {
		t.Fatal(err)
	}
}

func left(t *testing.T, home string) []string {
	t.Helper()
	var out []string
	filepath.WalkDir(home, func(p string, d os.DirEntry, err error) error {
		if err == nil && !d.IsDir() {
			rel, _ := filepath.Rel(home, p)
			out = append(out, filepath.ToSlash(rel))
		}
		return nil
	})
	sort.Strings(out)
	return out
}

func TestPruneRemovesDeadRunsOldEventsAndStaleMarkers(t *testing.T) {
	home := t.TempDir()
	fakeAlive(t, 10)
	now := fixed
	put(t, home, "run/10.json", now)                            // alive: kept
	put(t, home, "run/11.json", now)                            // dead: removed
	put(t, home, "run/touch-aaaaaaaaaaaa", now.Add(-time.Hour)) // recent: kept
	put(t, home, "run/touch-bbbbbbbbbbbb", now.Add(-25*time.Hour))
	put(t, home, "run/.tmp-123", now)            // not ours to judge: kept
	put(t, home, "run/notes.txt", now)           // not a record: kept
	put(t, home, "events/2026-10-28.jsonl", now) // today
	put(t, home, "events/2026-07-30.jsonl", now) // 90 days ago: kept
	put(t, home, "events/2026-07-29.jsonl", now) // 91 days ago: removed
	put(t, home, "events/2025-01-01.jsonl", now)
	put(t, home, "events/readme.jsonl", now) // not dated: kept
	if err := Prune(home, now); err != nil {
		t.Fatal(err)
	}
	want := []string{
		"events/2026-07-30.jsonl", "events/2026-10-28.jsonl", "events/readme.jsonl",
		"run/.tmp-123", "run/10.json", "run/notes.txt", "run/touch-aaaaaaaaaaaa",
	}
	got := left(t, home)
	if len(got) != len(want) {
		t.Fatalf("left = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("left = %v, want %v", got, want)
		}
	}
}

func TestPruneDoesNothingWithoutTheHomeFolder(t *testing.T) {
	home := filepath.Join(t.TempDir(), "absent")
	if err := Prune(home, fixed); err != nil {
		t.Fatal(err)
	}
	if err := Prune("", fixed); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(home); !os.IsNotExist(err) {
		t.Fatalf("the home folder appeared: %v", err)
	}
}

func TestPruneKeepsTheGoldenHome(t *testing.T) {
	home := t.TempDir()
	writeGolden(home)
	fakeAlive(t, 4242)
	before := files(t, home)
	if err := Prune(home, fixed); err != nil {
		t.Fatal(err)
	}
	if after := files(t, home); len(after) != len(before) {
		t.Fatalf("pruned a fresh home: %v -> %v", keys(before), keys(after))
	}
}

func TestThisProcessIsAlive(t *testing.T) {
	if !alive(os.Getpid()) {
		t.Fatal("this process is not alive")
	}
	if alive(0) || alive(-1) {
		t.Fatal("pid 0 or -1 is alive")
	}
}
