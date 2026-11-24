package store

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/LeeSwallow/stickypane/internal/core/doc"
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

func TestANewFolderIsSeeded(t *testing.T) {
	s := newStore(t)
	var asked []string
	s.NewFolder = func(folder string) map[string][]byte {
		asked = append(asked, folder)
		if folder == "worktrees/a" {
			return map[string][]byte{"00-about.md": []byte("about\n")}
		}
		return nil
	}
	if err := s.Write("worktrees/a/notes.md", []byte("x\n")); err != nil {
		t.Fatal(err)
	}
	if got := read(t, s, "worktrees/a/00-about.md"); got != "about\n" {
		t.Fatalf("seed = %q", got)
	}
	// A folder that is there is not made, and not asked about again.
	write(t, s, "worktrees/a/00-about.md", "changed\n")
	if err := s.Apply("worktrees/a/notes.md", keepOp{}); err != nil {
		t.Fatal(err)
	}
	if err := s.WriteNew("worktrees/a/more.md", []byte("y\n")); err != nil {
		t.Fatal(err)
	}
	if got := read(t, s, "worktrees/a/00-about.md"); got != "changed\n" {
		t.Fatalf("seed rewritten: %q", got)
	}
	// Every way a note reaches a new folder asks.
	if err := s.Rewrite("b/x.md", func([]byte, bool) ([]byte, error) { return []byte("x"), nil }); err != nil {
		t.Fatal(err)
	}
	if err := s.WriteNew("c/x.md", []byte("x")); err != nil {
		t.Fatal(err)
	}
	if err := s.Move("b/x.md", "d/x.md"); err != nil {
		t.Fatal(err)
	}
	want := []string{"worktrees/a", "b", "c", "d"}
	if len(asked) != len(want) {
		t.Fatalf("asked = %v, want %v", asked, want)
	}
	for i := range want {
		if asked[i] != want[i] {
			t.Fatalf("asked = %v, want %v", asked, want)
		}
	}
}

type keepOp struct{}

func (keepOp) Apply(d doc.Document) (doc.Document, error) { return d, nil }
