//go:build !windows

package runner

import (
	"os"
	"path/filepath"
	"syscall"
	"testing"
	"time"

	"github.com/LeeSwallow/stickypane/internal/store"
)

// longScript makes a notes folder with a script that runs until killed.
func longScript(t *testing.T) *store.Store {
	t.Helper()
	dir := filepath.Join(t.TempDir(), store.DirName)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "long.sh"), []byte("exec sleep 60\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	return store.Open(dir)
}

func pgidOf(t *testing.T, r *Run) int {
	t.Helper()
	if r.cmd == nil || r.cmd.Process == nil {
		t.Fatalf("the script did not start: %v", r.err)
	}
	g, err := syscall.Getpgid(r.cmd.Process.Pid)
	if err != nil {
		t.Fatal(err)
	}
	return g
}

// A script run from the terminal stays in the board's process group: it
// hangs up with the terminal and may read it.
func TestStartKeepsTheBoardsProcessGroup(t *testing.T) {
	r, err := Start(longScript(t), "long.sh", time.Now)
	if err != nil {
		t.Fatal(err)
	}
	defer r.Wait()
	defer r.Kill()
	if g := pgidOf(t, r); g != syscall.Getpgrp() {
		t.Errorf("pgid = %d, want the board's %d", g, syscall.Getpgrp())
	}
}

// A script the overlay runs leads a group of its own, so Kill stops what it
// started too.
func TestStartGroupLeadsItsOwnGroup(t *testing.T) {
	r, err := StartGroup(longScript(t), "long.sh", time.Now)
	if err != nil {
		t.Fatal(err)
	}
	defer r.Wait()
	defer r.Kill()
	if g := pgidOf(t, r); g != r.cmd.Process.Pid {
		t.Errorf("pgid = %d, want its own %d", g, r.cmd.Process.Pid)
	}
}

// openFiles counts the board's open files, or skips where it cannot.
func openFiles(t *testing.T) int {
	t.Helper()
	fds, err := os.ReadDir("/dev/fd")
	if err != nil {
		t.Skip("no /dev/fd here")
	}
	return len(fds)
}

// The board lets go of the log as soon as the script has it: on Windows a
// handle left open keeps the log from being moved or removed.
func TestStartLeavesTheLogToTheScript(t *testing.T) {
	st := longScript(t)
	before := openFiles(t)
	r, err := Start(st, "long.sh", time.Now)
	if err != nil {
		t.Fatal(err)
	}
	defer r.Wait()
	defer r.Kill()
	if after := openFiles(t); after != before {
		t.Errorf("open files = %d after Start, want %d: the board still holds the log", after, before)
	}
}
