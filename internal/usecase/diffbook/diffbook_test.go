package diffbook

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// repo makes a repository on main with two worktrees, a and b, that both
// change shared.go; a also adds a.txt, b leaves a file uncommitted.
func repo(t *testing.T) (main, a, b string) {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("no git")
	}
	dir := t.TempDir()
	main, a, b = filepath.Join(dir, "repo"), filepath.Join(dir, "a"), filepath.Join(dir, "b")
	git := func(in string, args ...string) {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = in
		cmd.Env = append(noGitDir(), "GIT_AUTHOR_NAME=t", "GIT_AUTHOR_EMAIL=t@t", "GIT_COMMITTER_NAME=t", "GIT_COMMITTER_EMAIL=t@t")
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	write := func(in, name, text string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(in, name), []byte(text), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	os.Mkdir(main, 0o755)
	git(main, "init", "-q", "-b", "main")
	write(main, "shared.go", "package x\n")
	git(main, "add", "-A")
	git(main, "commit", "-q", "-m", "base")
	git(main, "worktree", "add", "-q", "-b", "feat/a", a)
	git(main, "worktree", "add", "-q", "-b", "feat/b", b)
	write(a, "shared.go", "package x\n\nvar a = 1\n")
	write(a, "a.txt", "a\n")
	git(a, "add", "-A")
	git(a, "commit", "-q", "-m", "a")
	write(b, "shared.go", "package x\n\nvar b = 2\n")
	git(b, "commit", "-q", "-am", "b")
	write(b, "wip.txt", "wip\n")
	return main, a, b
}

func TestReadListsWorktreesAndWhatTwoOfThemChanged(t *testing.T) {
	main, a, _ := repo(t)
	book, err := Read(main, "", "")
	if err != nil {
		t.Fatal(err)
	}
	o := book.Overview
	if len(o.Worktrees) != 3 || o.Chosen != "main" || !o.Worktrees[0].Current {
		t.Fatalf("Overview = %+v", o)
	}
	rows := map[string]int{}
	for i, w := range o.Worktrees {
		rows[w.Name] = i
		if w.Base != "main" {
			t.Errorf("%s compared with %q, want main (the main worktree's branch, no remote)", w.Name, w.Base)
		}
	}
	if w := o.Worktrees[rows["feat/a"]]; w.Files != 2 || w.Add != 3 {
		t.Errorf("feat/a = %+v", w)
	}
	if len(o.Overlaps) != 1 || o.Overlaps[0].Path != "shared.go" || strings.Join(o.Overlaps[0].By, ",") != "feat/a,feat/b" {
		t.Errorf("Overlaps = %+v", o.Overlaps)
	}
	// The main worktree changed nothing since main; its pages are none.
	if len(book.Changes) != 0 {
		t.Errorf("Changes of main = %+v", book.Changes)
	}
	// Chosen from another worktree: its pages are its changes.
	book, err = Read(main, "feat/a", "")
	if err != nil || book.Overview.Chosen != "feat/a" || len(book.Changes) != 2 {
		t.Fatalf("Read(feat/a) = %+v, %v", book, err)
	}
	text, _, err := book.Diff(book.Changes[1])
	if err != nil || !strings.Contains(text, "+var a = 1") {
		t.Errorf("Diff = %q, %v", text, err)
	}
	// From inside a worktree, that one is current and chosen by default.
	if book, _ := Read(a, "", ""); book.Overview.Chosen != "feat/a" {
		t.Errorf("from a, chosen = %q", book.Overview.Chosen)
	}
	// A name git no longer has falls back to the current worktree.
	if book, _ := Read(main, "feat/gone", ""); book.Overview.Chosen != "main" {
		t.Errorf("a gone name chose %q", book.Overview.Chosen)
	}
}

func TestUncommittedChangesComeAfterTheCommittedOnes(t *testing.T) {
	main, _, _ := repo(t)
	book, err := Read(main, "feat/b", "")
	if err != nil {
		t.Fatal(err)
	}
	if len(book.Changes) != 2 || book.Changes[0].Path != "shared.go" || book.Changes[1].Path != "wip.txt" || book.Changes[1].Where != "untracked" {
		t.Fatalf("Changes = %+v", book.Changes)
	}
}

func TestAChosenBaseIsUsedForEveryWorktree(t *testing.T) {
	main, _, _ := repo(t)
	book, err := Read(main, "feat/a", "feat/b")
	if err != nil {
		t.Fatal(err)
	}
	for _, w := range book.Overview.Worktrees {
		if w.Base != "feat/b" {
			t.Errorf("%s base %q", w.Name, w.Base)
		}
	}
	if book.Base != "feat/b" || len(book.Changes) != 2 {
		t.Errorf("book = %+v", book)
	}
	// A base git does not know is said in each row, not hidden.
	book, _ = Read(main, "", "no-such-branch")
	if book.Overview.Worktrees[1].Err == "" {
		t.Errorf("unknown base: %+v", book.Overview.Worktrees[1])
	}
}

// Without a branch to compare with (the main worktree's HEAD detached, no
// remote), the rows say there is no base, and the pages are the
// uncommitted changes alone.
func TestNoBase(t *testing.T) {
	main, _, b := repo(t)
	cmd := exec.Command("git", "-C", main, "checkout", "-q", "--detach")
	cmd.Env = noGitDir()
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("%v %s", err, out)
	}
	book, err := Read(b, "", "")
	if err != nil {
		t.Fatal(err)
	}
	for _, w := range book.Overview.Worktrees {
		if w.Base != "" || w.Files != 0 {
			t.Errorf("%s = %+v", w.Name, w)
		}
	}
	if len(book.Changes) != 1 || book.Changes[0].Path != "wip.txt" {
		t.Errorf("Changes = %+v", book.Changes)
	}
}

func TestOutsideARepository(t *testing.T) {
	if _, err := Read(t.TempDir(), "", ""); err == nil {
		t.Error("no error outside a repository")
	}
}

// noGitDir is the environment without git's GIT_ variables, which a hook
// or rebase --exec sets: the git a test runs must work in its own folder,
// never in the repository the tests were started from.
func noGitDir() []string {
	var env []string
	for _, kv := range os.Environ() {
		if !strings.HasPrefix(kv, "GIT_") {
			env = append(env, kv)
		}
	}
	return env
}
