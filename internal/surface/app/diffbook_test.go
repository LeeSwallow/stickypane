package app

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// gitRepo makes the project around the notes folder dir a repository on
// main with two worktrees, feat/a and feat/b, that both change shared.go;
// feat/a also adds a.txt. The notes folder is not part of it.
func gitRepo(t *testing.T, dir string) {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("no git")
	}
	project := filepath.Dir(dir)
	git := func(in string, args ...string) {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = in
		cmd.Env = append(os.Environ(), "GIT_AUTHOR_NAME=t", "GIT_AUTHOR_EMAIL=t@t", "GIT_COMMITTER_NAME=t", "GIT_COMMITTER_EMAIL=t@t")
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	put := func(in, name, text string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(in, name), []byte(text), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	git(project, "init", "-q", "-b", "main")
	put(project, ".gitignore", ".sticky/\nwt-*/\n")
	put(project, "shared.go", "package x\n")
	git(project, "add", "-A")
	git(project, "commit", "-q", "-m", "base")
	a, b := filepath.Join(project, "wt-a"), filepath.Join(project, "wt-b")
	git(project, "worktree", "add", "-q", "-b", "feat/a", a)
	git(project, "worktree", "add", "-q", "-b", "feat/b", b)
	put(a, "shared.go", "package x\n\nvar a = 1\n")
	put(a, "a.txt", "a\n")
	git(a, "add", "-A")
	git(a, "commit", "-q", "-m", "a")
	put(b, "shared.go", "package x\n\nvar b = 2\n")
	git(b, "commit", "-q", "-am", "b")
}

const diffNote = "---\ntype: diff\nopen: true\nsize: page\n---\n"

func TestADiffNoteIsABookOfTheWorktreesChanges(t *testing.T) {
	t.Parallel()
	m, dir := newModel(t, map[string]string{"changes.md": diffNote})
	gitRepo(t, dir)
	press(m, "r")
	s := screen(m)
	for _, want := range []string{"feat/a  main…  2 files  +3 −0", "feat/b  main…  1 file  +2 −0", "⚠ shared.go", "changed in feat/a · feat/b", "1/1"} {
		if !strings.Contains(s, want) {
			t.Errorf("the list page lacks %q:\n%s", want, s)
		}
	}
	// Zoomed, j and k choose a worktree and enter shows its files.
	press(m, "z", "j", "enter")
	if v := viewsOf(t, dir)["changes.md"]; v.Worktree != "feat/a" {
		t.Fatalf("the choice is kept in sticky.json: %+v", v)
	}
	s = screen(m)
	if !strings.Contains(s, "A a.txt  +1 −0") || !strings.Contains(s, "2/3") {
		t.Fatalf("enter turns to the first changed file:\n%s", s)
	}
	press(m, ".")
	if s = screen(m); !strings.Contains(s, "M shared.go  +2 −0") || !strings.Contains(s, "+ var a = 1") || !strings.Contains(s, "3/3") {
		t.Fatalf(". turns to the next file:\n%s", s)
	}
	// A page git made has no file: it is not edited, moved or deleted.
	press(m, "esc", "x")
	if s = screen(m); !strings.Contains(s, "read only") || !strings.Contains(s, "3/3") {
		t.Fatalf("x on a file page is refused:\n%s", s)
	}
	if got := readFile(t, dir, "changes.md"); got != diffNote {
		t.Errorf("the note was written: %q", got)
	}
	// b sets the base, kept in sticky.json too.
	press(m, ",", ",", "b")
	typeText(m, "feat/b")
	press(m, "enter")
	if v := viewsOf(t, dir)["changes.md"]; v.Base != "feat/b" || v.Worktree != "feat/a" {
		t.Fatalf("b keeps the base: %+v", v)
	}
	if s = screen(m); !strings.Contains(s, "feat/a  feat/b…  2 files  +3 −0") {
		t.Errorf("the list compares with the base chosen:\n%s", s)
	}
}

// worktree: in the front matter pins one worktree: no list, its files only.
func TestAPinnedWorktreeSkipsTheList(t *testing.T) {
	t.Parallel()
	m, dir := newModel(t, map[string]string{"changes.md": "---\ntype: diff\nopen: true\nworktree: feat/b\n---\n"})
	gitRepo(t, dir)
	press(m, "r")
	s := screen(m)
	if !strings.Contains(s, "M shared.go  +2 −0") || !strings.Contains(s, "1/1") || strings.Contains(s, "⚠") {
		t.Fatalf("a pinned worktree shows its files alone:\n%s", s)
	}
}

// Outside a repository the list says there is nothing to show.
func TestADiffNoteOutsideARepository(t *testing.T) {
	t.Parallel()
	m, _ := newModel(t, map[string]string{"changes.md": diffNote})
	if s := screen(m); !strings.Contains(s, "no git worktrees here") {
		t.Fatalf("outside a repository:\n%s", s)
	}
}
