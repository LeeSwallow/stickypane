package api

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/LeeSwallow/stickypane/internal/store"
)

func has(dir, name string) bool {
	_, err := os.Lstat(filepath.Join(dir, filepath.FromSlash(name)))
	return err == nil
}

func TestRemoveGoesToTheTrashAndRestoreBringsItBack(t *testing.T) {
	a, dir := newAPI(t, map[string]string{"plan.md": "the plan\n"})
	got, err := a.Remove("plan")
	if err != nil || has(dir, "plan.md") || !has(dir, ".trash/plan.md") {
		t.Fatalf("Remove = %q, %v", got, err)
	}
	if !strings.Contains(got, ".trash/plan.md") || !strings.Contains(got, "restore plan") {
		t.Errorf("the answer should say where the note went and how to get it back: %q", got)
	}
	if got, err := a.Restore("plan"); err != nil || got != "restored plan.md" || read(t, dir, "plan.md") != "the plan\n" {
		t.Errorf("Restore = %q, %v", got, err)
	}
	if _, err := a.Restore("plan"); err == nil {
		t.Error("there is nothing left to restore")
	}
	if _, err := a.Remove("missing"); err == nil {
		t.Error("removing what is not there is an error")
	}
}

func TestArchiveAndRemoveTakeABookByItsFolder(t *testing.T) {
	a, dir := newAPI(t, map[string]string{"plan.md": "the plan\n"})
	if _, err := a.Write("docs/guide", Options{}, []byte("guide\n")); err != nil {
		t.Fatal(err)
	}
	if got, err := a.Archive("plan"); err != nil || got != "moved plan.md to archive/plan.md" || !has(dir, "archive/plan.md") {
		t.Errorf("Archive = %q, %v", got, err)
	}
	if _, err := a.Remove("docs"); err != nil || has(dir, "docs") || !has(dir, ".trash/docs/guide.md") {
		t.Errorf("a book goes to the trash whole: %v", err)
	}
	if _, err := a.Restore("docs"); err != nil || read(t, dir, "docs/guide.md") != "guide\n" {
		t.Errorf("and comes back whole: %v", err)
	}
}

func TestMove(t *testing.T) {
	a, dir := newAPI(t, map[string]string{"plan.md": "the plan\n", "build.log": "x\n", "other.md": "other\n"})
	if err := os.MkdirAll(filepath.Join(dir, "docs"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "docs", "p.md"), []byte("p\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	steps := []struct{ name, to, want, file string }{
		{"plan", "docs", "moved plan.md to docs/plan.md", "docs/plan.md"},                 // into a book
		{"docs/plan", ".", "moved docs/plan.md to plan.md", "plan.md"},                    // out to the top level
		{"plan", "roadmap", "moved plan.md to roadmap.md", "roadmap.md"},                  // a new name
		{"build.log", "ci", "moved build.log to ci.log", "ci.log"},                        // a new name keeps the kind of file
		{"roadmap", "notes/", "moved roadmap.md to notes/roadmap.md", "notes/roadmap.md"}, // a folder that is not there yet
		{"docs", "manual", "moved docs to manual", "manual/p.md"},                         // a book under a new name
	}
	for _, s := range steps {
		if got, err := a.Move(s.name, s.to); err != nil || got != s.want || !has(dir, s.file) {
			t.Errorf("Move(%q, %q) = %q, %v", s.name, s.to, got, err)
		}
	}
	if _, err := a.Move("other", "ci.log"); err == nil || read(t, dir, "ci.log") != "x\n" {
		t.Errorf("Move must not write over another note: %v", err)
	}
	for _, bad := range [][2]string{{"other", "../out"}, {"other", "a/b/c"}, {"other", ""}, {"missing", "x"}, {"other", "archive/x"}} {
		if _, err := a.Move(bad[0], bad[1]); err == nil {
			t.Errorf("Move(%q, %q) should fail", bad[0], bad[1])
		}
	}
	if !has(dir, "other.md") {
		t.Error("a refused move leaves the note where it was")
	}
}

func TestMovingANoteTakesItsArrangementAlong(t *testing.T) {
	a, dir := newAPI(t, map[string]string{"plan.md": "the plan\n"})
	if _, err := a.Set("plan", []string{"open=true", "size=half"}); err != nil {
		t.Fatal(err)
	}
	if _, err := a.Move("plan", "roadmap"); err != nil {
		t.Fatal(err)
	}
	v, _ := store.Open(dir).Views()
	if n := v["roadmap.md"]; n.Open == nil || !*n.Open || n.Size != "half" {
		t.Errorf("a renamed note keeps where it was on the screen: %+v", v)
	}
	if _, old := v["plan.md"]; old {
		t.Errorf("the old name should be gone from sticky.json: %+v", v)
	}
}

func TestLinkShowsAFileOfTheProjectWithoutCopyingIt(t *testing.T) {
	a, dir := newAPI(t, nil)
	root := filepath.Dir(dir)
	if err := os.WriteFile(filepath.Join(root, "README.md"), []byte("# Readme\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "docs"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "docs", "guide.md"), []byte("guide\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := a.Link(filepath.Join(root, "README.md"), "")
	if err != nil {
		t.Skip("symlinks are not available:", err)
	}
	if got != "linked README.md to ../README.md" || read(t, dir, "README.md") != "# Readme\n" {
		t.Errorf("Link = %q", got)
	}
	if target, _ := os.Readlink(filepath.Join(dir, "README.md")); target != filepath.Join("..", "README.md") {
		t.Errorf("the link should be relative so the project can be moved: %q", target)
	}
	v, _ := store.Open(dir).Views()
	if n := v["README.md"]; n.Open == nil || !*n.Open {
		t.Errorf("a linked note is opened: %+v", v)
	}
	if got, err := a.Link(filepath.Join(root, "docs"), "handbook"); err != nil || got != "linked handbook to ../docs" || read(t, dir, "handbook/guide.md") != "guide\n" {
		t.Errorf("a folder is linked as a book, under a name of its own if one is given: %q, %v", got, err)
	}
	if infos, _ := a.List(); len(infos) != 2 || infos[1].Name != "handbook/guide.md" {
		t.Errorf("List = %+v", infos)
	}
	if _, err := a.Link(filepath.Join(root, "README.md"), ""); err == nil {
		t.Error("a name that is taken is refused")
	}
	if _, err := a.Link(filepath.Join(root, "missing.md"), ""); err == nil {
		t.Error("a file that is not there cannot be linked")
	}
	if err := os.WriteFile(filepath.Join(root, "logo.png"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := a.Link(filepath.Join(root, "logo.png"), ""); err == nil || !strings.Contains(err.Error(), ".md") {
		t.Errorf("a file the board cannot show is refused, saying what it can: %v", err)
	}
}

func TestALinkedNoteIsNamedWithoutTouchingItsFile(t *testing.T) {
	a, dir := newAPI(t, nil)
	root := filepath.Dir(dir)
	if err := os.WriteFile(filepath.Join(root, "README.md"), []byte("# Readme\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := a.Link(filepath.Join(root, "README.md"), ""); err != nil {
		t.Skip("symlinks are not available:", err)
	}
	if got, err := a.Set("README.md", []string{"title=Read me", "size=half"}); err != nil || got != "README.md: set title, size" {
		t.Fatalf("Set = %q, %v", got, err)
	}
	if b, _ := os.ReadFile(filepath.Join(root, "README.md")); string(b) != "# Readme\n" {
		t.Errorf("the project's file must not get front matter: %q", b)
	}
	v, _ := store.Open(dir).Views()
	if v["README.md"].Title != "Read me" {
		t.Errorf("the name goes to sticky.json: %+v", v)
	}
}
