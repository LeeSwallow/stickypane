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

// A .git file that git's records do not point back to is not believed: a
// folder unpacked from an archive must not send the board to another
// repository.
func TestAForgedDotGitFileLeadsNowhere(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("no git")
	}
	dir := t.TempDir()
	repo, wt, fake := filepath.Join(dir, "repo"), filepath.Join(dir, "wt"), filepath.Join(dir, "fake")
	os.Mkdir(repo, 0o755)
	os.Mkdir(fake, 0o755)
	for _, args := range [][]string{{"init", "-q"}, {"commit", "-q", "--allow-empty", "-m", "first"}, {"worktree", "add", "-q", "-b", "feat", wt}} {
		cmd := exec.Command("git", args...)
		cmd.Dir = repo
		cmd.Env = append(os.Environ(), "GIT_AUTHOR_NAME=t", "GIT_AUTHOR_EMAIL=t@t", "GIT_COMMITTER_NAME=t", "GIT_COMMITTER_EMAIL=t@t")
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	// The fake copies the real worktree's .git line: git reads it, but the
	// worktree's record names wt/.git, not fake/.git.
	b, _ := os.ReadFile(filepath.Join(wt, ".git"))
	os.WriteFile(filepath.Join(fake, ".git"), b, 0o644)
	if m := Main(fake); m != "" {
		t.Fatalf("Main of a forged .git = %q, want none", m)
	}
	// One that names a folder outside any worktrees/ is not believed either.
	os.WriteFile(filepath.Join(fake, ".git"), []byte("gitdir: "+filepath.Join(repo, ".git")+"\n"), 0o644)
	if m := Main(fake); m != "" {
		t.Fatalf("Main of a .git naming the common dir = %q, want none", m)
	}
	if m := Main(wt); m == "" {
		t.Fatal("the real worktree lost its main")
	}
}

// git sets GIT_DIR and its kin for a hook or rebase --exec. A stickypane
// run from there reads the folder it was given, not the caller's
// repository.
func TestTheCallersGitDirIsIgnored(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("no git")
	}
	dir := t.TempDir()
	repo, decoy := filepath.Join(dir, "repo"), filepath.Join(dir, "decoy")
	for _, d := range []string{repo, decoy} {
		os.Mkdir(d, 0o755)
		if out, err := exec.Command("git", "-C", d, "init", "-q", "-b", "main").CombinedOutput(); err != nil {
			t.Fatalf("git init: %v\n%s", err, out)
		}
	}
	t.Setenv("GIT_DIR", filepath.Join(decoy, ".git"))
	t.Setenv("GIT_WORK_TREE", decoy)
	common, err := CommonDir(repo)
	real := func(p string) string { r, _ := filepath.EvalSymlinks(p); return r }
	if err != nil || real(common) != real(filepath.Join(repo, ".git")) {
		t.Fatalf("CommonDir = %q, %v; want the repository's own", common, err)
	}
}
