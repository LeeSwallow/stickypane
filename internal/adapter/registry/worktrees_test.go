package registry

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

// goldenWorktrees is a home where one repository's worktrees were followed:
// feat-a opened, moved, and closed; feat-b opened. The desktop app's tests
// read testdata/home, so these records are pinned apart from it.
const goldenWorktrees = "testdata/worktrees"

const board = "/work/demo/.git"

func writeWorktrees(home string) {
	w := fixedWriter(home, 4242)
	a := Worktree{Name: "feat-a", Path: "/work/wt/feat-a", Branch: "feat/a", State: StateOpen}
	b := Worktree{Name: "feat-b", Path: "/work/wt/feat-b", Branch: "feat/b", State: StateOpen}
	w.UpdateWorktrees(board, func(old []Worktree) ([]Worktree, []Event) {
		return []Worktree{a}, []Event{{Type: WorktreeOpen, Project: "/work/demo", Name: a.Name, Path: a.Path}}
	})
	moved := a
	moved.Path = "/work/elsewhere/feat-a"
	w.UpdateWorktrees(board, func(old []Worktree) ([]Worktree, []Event) {
		return []Worktree{moved, b}, []Event{
			{Type: WorktreeMove, Project: "/work/demo", Name: a.Name, Path: moved.Path, From: a.Path},
			{Type: WorktreeOpen, Project: "/work/demo", Name: b.Name, Path: b.Path},
		}
	})
	closed := moved
	closed.State = StateClosed
	w.UpdateWorktrees(board, func(old []Worktree) ([]Worktree, []Event) {
		return []Worktree{closed, b}, []Event{{Type: WorktreeClose, Project: "/work/demo", Name: a.Name, Path: moved.Path}}
	})
}

func TestTheWorktreesGolden(t *testing.T) {
	home := appHome(t)
	writeWorktrees(home)
	matchGolden(t, home, goldenWorktrees)
}

func TestUpdateWorktreesHandsOverTheRecord(t *testing.T) {
	home := appHome(t)
	writeWorktrees(home)
	var got []Worktree
	fixedWriter(home, 1).UpdateWorktrees(board, func(old []Worktree) ([]Worktree, []Event) {
		got = old
		return old, nil
	})
	want := []Worktree{
		{Name: "feat-a", Path: "/work/elsewhere/feat-a", Branch: "feat/a", State: StateClosed},
		{Name: "feat-b", Path: "/work/wt/feat-b", Branch: "feat/b", State: StateOpen},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("old = %+v, want %+v", got, want)
	}
	if other := (Writer{Home: home}).Worktrees("/other/.git"); other != nil {
		t.Fatalf("another board's record = %+v", other)
	}
}

// Without the app's file nothing is written, but change still runs, with
// no record, so that a caller does its other work the same way.
func TestWorktreesWithoutTheAppsFile(t *testing.T) {
	home := t.TempDir()
	ran := false
	fixedWriter(home, 1).UpdateWorktrees(board, func(old []Worktree) ([]Worktree, []Event) {
		ran = old == nil
		return []Worktree{{Name: "x", State: StateOpen}}, []Event{{Type: WorktreeOpen, Name: "x"}}
	})
	if !ran {
		t.Fatal("change did not run, or was handed a record")
	}
	if ents, _ := os.ReadDir(home); len(ents) != 0 {
		t.Fatalf("wrote %v", ents)
	}
	if _, err := os.Stat(filepath.Join(home, WorktreesFile)); err == nil {
		t.Fatal("worktrees.json written")
	}
}
