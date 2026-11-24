package store

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// repoWithWorktree makes a repository main and a second worktree of it, wt,
// beside it rather than under it, so that only git can lead from one to
// the other.
func repoWithWorktree(t *testing.T) (main, wt string) {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("no git")
	}
	dir := t.TempDir()
	main, wt = filepath.Join(dir, "main"), filepath.Join(dir, "wt")
	os.Mkdir(main, 0o755)
	for _, args := range [][]string{
		{"init", "-q"},
		{"commit", "-q", "--allow-empty", "-m", "first"},
		{"worktree", "add", "-q", "-b", "feat", wt},
	} {
		cmd := exec.Command("git", args...)
		cmd.Dir = main
		cmd.Env = append(os.Environ(), "GIT_AUTHOR_NAME=t", "GIT_AUTHOR_EMAIL=t@t", "GIT_COMMITTER_NAME=t", "GIT_COMMITTER_EMAIL=t@t")
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	return main, wt
}

func same(a, b string) bool {
	fa, err1 := os.Stat(a)
	fb, err2 := os.Stat(b)
	return err1 == nil && err2 == nil && os.SameFile(fa, fb)
}

func TestAWorktreeUsesTheMainWorktreesBoard(t *testing.T) {
	main, wt := repoWithWorktree(t)
	board := filepath.Join(main, DirName)
	os.Mkdir(board, 0o755)
	sub := filepath.Join(wt, "sub")
	os.Mkdir(sub, 0o755)
	for _, start := range []string{wt, sub} {
		got, err := Resolve(start)
		if err != nil || !same(got, board) {
			t.Errorf("Resolve(%s) = %q, %v; want %s", start, got, err, board)
		}
	}
}

func TestAWorktreesOwnBoardWins(t *testing.T) {
	main, wt := repoWithWorktree(t)
	os.Mkdir(filepath.Join(main, DirName), 0o755)
	own := filepath.Join(wt, DirName)
	os.Mkdir(own, 0o755)
	if got, err := Resolve(wt); err != nil || got != own {
		t.Fatalf("Resolve = %q, %v; want %s", got, err, own)
	}
}

func TestAWorktreeWithoutAMainBoardFindsNone(t *testing.T) {
	_, wt := repoWithWorktree(t)
	if _, err := Resolve(wt); err != ErrNotFound {
		t.Fatalf("err = %v, want ErrNotFound", err)
	}
}
