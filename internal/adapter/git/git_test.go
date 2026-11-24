package git

import (
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"testing"
)

func TestParseReadsEveryKindOfWorktree(t *testing.T) {
	b, err := os.ReadFile("testdata/list.txt")
	if err != nil {
		t.Fatal(err)
	}
	want := []Worktree{
		{Path: "/work/demo", Branch: "main"},
		{Path: "/work/demo/.claude/worktrees/chart", Branch: "worktree-chart", Locked: true},
		{Path: "/work/wt/feat-a", Branch: "feat/a"},
		{Path: "/work/wt/gone", Branch: "gone", Prunable: true},
		{Path: "/work/wt/try it"},
	}
	if got := Parse(b); !reflect.DeepEqual(got, want) {
		t.Fatalf("Parse =\n%+v\nwant\n%+v", got, want)
	}
}

func TestParseReadsABareMain(t *testing.T) {
	got := Parse([]byte("worktree /srv/repo.git\nbare\n\nworktree /work/a\nHEAD 1\nlocked\n"))
	want := []Worktree{{Path: "/srv/repo.git", Bare: true}, {Path: "/work/a", Locked: true}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Parse = %+v, want %+v", got, want)
	}
}

// The one test that runs git: a repository, a second worktree, and what
// the package reads of them.
func TestReadsARealRepository(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("no git")
	}
	dir := t.TempDir()
	repo, wt := filepath.Join(dir, "repo"), filepath.Join(dir, "wt one")
	run := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = repo
		cmd.Env = append(os.Environ(), "GIT_AUTHOR_NAME=t", "GIT_AUTHOR_EMAIL=t@t", "GIT_COMMITTER_NAME=t", "GIT_COMMITTER_EMAIL=t@t")
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	os.Mkdir(repo, 0o755)
	run("init", "-q", "-b", "main")
	run("commit", "-q", "--allow-empty", "-m", "first")
	run("worktree", "add", "-q", "-b", "feat/x", wt)

	list, err := Worktrees(wt)
	if err != nil {
		t.Fatal(err)
	}
	real := func(p string) string { r, _ := filepath.EvalSymlinks(p); return r }
	if len(list) != 2 || real(list[0].Path) != real(repo) || real(list[1].Path) != real(wt) || list[1].Branch != "feat/x" {
		t.Fatalf("Worktrees = %+v", list)
	}
	if m := Main(wt); real(m) != real(repo) {
		t.Fatalf("Main = %q, want %q", m, repo)
	}
	common, err := CommonDir(wt)
	if err != nil || real(common) != real(filepath.Join(repo, ".git")) {
		t.Fatalf("CommonDir = %q, %v", common, err)
	}
	if Main(dir) != "" {
		t.Fatal("Main outside a repository is not empty")
	}
}
