package bridge

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/LeeSwallow/stickypane/internal/adapter/store"
	"github.com/LeeSwallow/stickypane/internal/usecase/worktree"
)

// note.arrange keeps a book's paging, as the terminal's v does, and
// board.load says it.
func TestBridgeArrangeSetsABooksPaging(t *testing.T) {
	dir := board(t, map[string]string{"t/docs/a.md": "a\n", "t/docs/b.md": "b\n", "plan.md": plan})
	s := serve(t, dir)
	s.ok("note.arrange", map[string]any{"note": "t/docs", "paging": "single"}, nil)
	if v, _ := store.Open(dir).Views(); v["t/docs"].Paging != store.PagingSingle {
		t.Fatalf("paging is kept in the book's view: %+v", v["t/docs"])
	}
	var b loadedBoard
	s.ok("board.load", nil, &b)
	if got := b.note(t, "t/docs").Paging; got != store.PagingSingle {
		t.Errorf("board.load says paging %q", got)
	}
	if got := b.note(t, "plan.md").Paging; got != "" {
		t.Errorf("a note that is no book has no paging: %q", got)
	}
	// A page is arranged by its book; "" shows every page again.
	s.ok("note.arrange", map[string]any{"note": "t/docs/b.md", "paging": ""}, nil)
	if v, _ := store.Open(dir).Views(); v["t/docs"].Paging != "" {
		t.Errorf("paging is taken back out: %+v", v["t/docs"])
	}
	for name, p := range map[string]map[string]any{
		"not a book": {"note": "plan.md", "paging": "single"},
		"two":        {"note": "t/docs", "paging": "two"},
	} {
		if r := s.call("note.arrange", p); r.code != CodeBadRequest {
			t.Errorf("%s: code = %q, want bad_request", name, r.code)
		}
	}
}

// A book of worktrees/ says the state of its worktree, worked out as
// `stickypane worktree list` works it out; other books say none.
func TestBridgeLoadSaysAWorktreeBooksState(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("no git")
	}
	form := "---\ntype: form\n---\nok?\n\n[ OK ]\n"
	dir := board(t, map[string]string{
		"worktrees/run/plan.md":               "plan\n",
		"worktrees/ask/q.md":                  form,
		"worktrees/idle/plan.md":              "plan\n",
		"worktrees/gone/" + worktree.Overview: "---\nbranch: gone\nstate: closed\nclosed: 2026-11-24 09:30\n---\n",
		"worktrees/gone/q.md":                 form,
		"t/docs/a.md":                         "a\n",
		"t/docs/b.md":                         "b\n",
	})
	root := filepath.Dir(dir)
	runGit := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = root
		cmd.Env = append(os.Environ(), "GIT_AUTHOR_NAME=t", "GIT_AUTHOR_EMAIL=t@t", "GIT_COMMITTER_NAME=t", "GIT_COMMITTER_EMAIL=t@t")
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	runGit("init", "-q", "-b", "main")
	runGit("commit", "-q", "--allow-empty", "-m", "first")
	wt := filepath.Join(t.TempDir(), "wt")
	for _, name := range []string{"run", "ask", "idle"} {
		runGit("worktree", "add", "-q", "-b", name, filepath.Join(wt, name))
	}
	runGit("worktree", "lock", filepath.Join(wt, "run"))
	runGit("worktree", "lock", filepath.Join(wt, "ask"))

	s := serve(t, dir)
	var b loadedBoard
	s.ok("board.load", nil, &b)
	for name, want := range map[string]struct {
		state   string
		waiting int
	}{
		"worktrees/run":  {"running", 0},
		"worktrees/ask":  {"waiting", 1}, // locked too, but the person is what it waits for
		"worktrees/idle": {"open", 0},
		"worktrees/gone": {"closed", 1},
		"t/docs":         {"", 0},
	} {
		if n := b.note(t, name); n.State != want.state || n.Waiting != want.waiting {
			t.Errorf("%s: state %q waiting %d, want %q %d", name, n.State, n.Waiting, want.state, want.waiting)
		}
	}
	if _, err := os.Stat(filepath.Join(dir, "worktrees", "run", worktree.Overview)); err == nil {
		t.Error("board.load wrote an overview: it only reads")
	}
}
