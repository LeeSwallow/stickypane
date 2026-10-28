//go:build !windows

package runner

import (
	"os"
	"path/filepath"
	"strconv"
	"syscall"
	"testing"
	"time"

	"github.com/LeeSwallow/stickypane/internal/adapter/store"
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

// A script the app runs leads a group of its own, so Kill stops what it
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

// holds says whether the board has the file open: whether any of its open
// files is that file. It skips where it cannot list them.
func holds(t *testing.T, name string) bool {
	t.Helper()
	var want syscall.Stat_t
	if err := syscall.Stat(name, &want); err != nil {
		t.Fatal(err)
	}
	for _, dir := range []string{"/proc/self/fd", "/dev/fd"} {
		fds, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, e := range fds {
			fd, err := strconv.Atoi(e.Name())
			if err != nil {
				continue
			}
			// Fstat asks the open file itself; a file closed since the
			// listing, or one that is no file (a pidfd, a pipe), is not
			// the log.
			var got syscall.Stat_t
			if syscall.Fstat(fd, &got) == nil && got.Dev == want.Dev && got.Ino == want.Ino {
				return true
			}
		}
		return false
	}
	t.Skip("no list of open files here")
	return false
}

// The board lets go of the log as soon as the script has it: on Windows a
// handle left open keeps the log from being moved or removed.
func TestStartLeavesTheLogToTheScript(t *testing.T) {
	st := longScript(t)
	log := filepath.Join(st.Dir, LogOf("long.sh"))
	f, err := os.Open(filepath.Join(st.Dir, "long.sh"))
	if err != nil {
		t.Fatal(err)
	}
	if !holds(t, f.Name()) {
		t.Fatal("holds does not see a file the board has open")
	}
	f.Close()
	r, err := Start(st, "long.sh", time.Now)
	if err != nil {
		t.Fatal(err)
	}
	defer r.Wait()
	defer r.Kill()
	if holds(t, log) {
		t.Error("the board still holds the log after Start")
	}
}
