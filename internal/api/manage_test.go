package api

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
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
		t.Fatalf("Remove = %+v, %v", got, err)
	}
	if got != (Moved{From: "plan.md", To: ".trash/plan.md"}) {
		t.Errorf("the answer should say where the note went: %+v", got)
	}
	if got, err := a.Restore("plan"); err != nil || got != (Moved{From: ".trash/plan.md", To: "plan.md"}) || read(t, dir, "plan.md") != "the plan\n" {
		t.Errorf("Restore = %+v, %v", got, err)
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
	if got, err := a.Archive("plan"); err != nil || got != (Moved{From: "plan.md", To: "archive/plan.md"}) || !has(dir, "archive/plan.md") {
		t.Errorf("Archive = %+v, %v", got, err)
	}
	if _, err := a.Remove("docs"); err != nil || has(dir, "docs") || !has(dir, ".trash/docs/guide.md") {
		t.Errorf("a book goes to the trash whole: %v", err)
	}
	if _, err := a.Restore("docs"); err != nil || read(t, dir, "docs/guide.md") != "guide\n" {
		t.Errorf("and comes back whole: %v", err)
	}
}

// A book in a tab is a book too: archived, trashed and restored whole, by
// its folder's name, and a book whose name has a dot comes back by it.
func TestABookInATabIsTakenByItsFolder(t *testing.T) {
	a, dir := newAPI(t, map[string]string{"x.md": "x\n"})
	for _, f := range []string{"deploy/docs/guide.md", "deploy/v1.2/notes.md"} {
		if err := os.MkdirAll(filepath.Dir(filepath.Join(dir, f)), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, f), []byte("g\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if got, err := a.Archive("deploy/docs"); err != nil || got.From != "deploy/docs" || !has(dir, got.To+"/guide.md") {
		t.Errorf("Archive = %+v, %v", got, err)
	}
	if got, err := a.Remove("deploy/v1.2"); err != nil || got.To != ".trash/deploy/v1.2" || has(dir, "deploy/v1.2") {
		t.Fatalf("Remove = %+v, %v", got, err)
	}
	if _, err := a.Restore("deploy/v1.2"); err != nil || !has(dir, "deploy/v1.2/notes.md") {
		t.Errorf("a book in a tab comes back whole: %v", err)
	}
	if _, err := a.Restore("v9.9"); !errors.Is(err, fs.ErrNotExist) || !strings.Contains(err.Error(), "there is no v9.9 in the trash") {
		t.Errorf("a book that was never trashed: %v", err)
	}
	if got, err := a.Set("deploy/v1.2", []string{"color=blue", "title=Release"}); err != nil || !reflect.DeepEqual(got, Keyed{Note: "deploy/v1.2", Set: []string{"color", "title"}}) {
		t.Errorf("Set on a book in a tab = %+v, %v", got, err)
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
	steps := []struct {
		name, to string
		want     Moved
		file     string
	}{
		{"plan", "docs", Moved{"plan.md", "docs/plan.md"}, "docs/plan.md"},                 // into a book
		{"docs/plan", ".", Moved{"docs/plan.md", "plan.md"}, "plan.md"},                    // out to the top level
		{"plan", "roadmap", Moved{"plan.md", "roadmap.md"}, "roadmap.md"},                  // a new name
		{"build.log", "ci", Moved{"build.log", "ci.log"}, "ci.log"},                        // a new name keeps the kind of file
		{"roadmap", "notes/", Moved{"roadmap.md", "notes/roadmap.md"}, "notes/roadmap.md"}, // a folder that is not there yet
		{"docs", "manual", Moved{"docs", "manual"}, "manual/p.md"},                         // a book under a new name
	}
	for _, s := range steps {
		if got, err := a.Move(s.name, s.to); err != nil || got != s.want || !has(dir, s.file) {
			t.Errorf("Move(%q, %q) = %+v, %v", s.name, s.to, got, err)
		}
	}
	if _, err := a.Move("other", "ci.log"); err == nil || read(t, dir, "ci.log") != "x\n" {
		t.Errorf("Move must not write over another note: %v", err)
	}
	for _, bad := range [][2]string{{"other", "../out"}, {"other", "a/b/c/d"}, {"other", ""}, {"missing", "x"}, {"other", "archive/x"}} {
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
	if n := v["roadmap.md"]; n.Open == nil || !*n.Open || n.Size != "1/2" {
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
	if got != (Linked{Note: "README.md", Target: "../README.md"}) || read(t, dir, "README.md") != "# Readme\n" {
		t.Errorf("Link = %+v", got)
	}
	if target, _ := os.Readlink(filepath.Join(dir, "README.md")); target != filepath.Join("..", "README.md") {
		t.Errorf("the link should be relative so the project can be moved: %q", target)
	}
	v, _ := store.Open(dir).Views()
	if n := v["README.md"]; n.Open == nil || !*n.Open {
		t.Errorf("a linked note is opened: %+v", v)
	}
	if got, err := a.Link(filepath.Join(root, "docs"), "handbook"); err != nil || got != (Linked{Note: "handbook", Target: "../docs"}) || read(t, dir, "handbook/guide.md") != "guide\n" {
		t.Errorf("a folder is linked as a book, under a name of its own if one is given: %+v, %v", got, err)
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
	if got, err := a.Set("README.md", []string{"title=Read me", "size=half"}); err != nil || !reflect.DeepEqual(got, Keyed{Note: "README.md", Set: []string{"title", "size"}}) {
		t.Fatalf("Set = %+v, %v", got, err)
	}
	if b, _ := os.ReadFile(filepath.Join(root, "README.md")); string(b) != "# Readme\n" {
		t.Errorf("the project's file must not get front matter: %q", b)
	}
	v, _ := store.Open(dir).Views()
	if v["README.md"].Title != "Read me" {
		t.Errorf("the name goes to sticky.json: %+v", v)
	}
}

func TestShowAndHide(t *testing.T) {
	a, dir := newAPI(t, map[string]string{"plan.md": "the plan\n", "build.log": "x\n"})
	root := filepath.Dir(dir)
	if err := os.WriteFile(filepath.Join(root, "README.md"), []byte("# Readme\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if got, err := a.Show("plan"); err != nil || got != (Shown{Note: "plan.md", Open: true}) {
		t.Fatalf("Show = %+v, %v", got, err)
	}
	if got, err := a.Show("build.log"); err != nil || got != (Shown{Note: "build.log", Open: true}) {
		t.Fatalf("Show of a log = %+v, %v", got, err)
	}
	v, _ := store.Open(dir).Views()
	if v["plan.md"].Open == nil || !*v["plan.md"].Open || v["build.log"].Open == nil || !*v["build.log"].Open {
		t.Errorf("Show opens the note on the screen: %+v", v)
	}
	if got, err := a.Hide("plan"); err != nil || got != (Shown{Note: "plan.md"}) {
		t.Fatalf("Hide = %+v, %v", got, err)
	}
	if v, _ := store.Open(dir).Views(); v["plan.md"].Open == nil || *v["plan.md"].Open {
		t.Errorf("Hide folds the note away: %+v", v)
	}
	// A path to a file of the project is linked onto the board and shown.
	got, err := a.Show(filepath.Join(root, "README.md"))
	if err != nil {
		t.Skip("symlinks are not available:", err)
	}
	if got != (Shown{Note: "README.md", Open: true, Linked: "../README.md"}) || !has(dir, "README.md") {
		t.Errorf("Show of a path = %+v", got)
	}
	if got, err := a.Show(filepath.Join(root, "README.md")); err != nil || got != (Shown{Note: "README.md", Open: true}) {
		t.Errorf("showing a linked file again just opens it: %+v, %v", got, err)
	}
	if _, err := a.Show("missing"); err == nil {
		t.Error("a note that is not there cannot be shown")
	}
	if _, err := a.Show(filepath.Join(root, "logo.png")); err == nil {
		t.Error("a file that is not there cannot be shown")
	}
}

func TestShowSwitchesTheBoardToTheNotesTab(t *testing.T) {
	a, dir := newAPI(t, map[string]string{"plan.md": "root\n"})
	if _, err := a.Write("deploy/notes", Options{}, []byte("deploy\n")); err != nil {
		t.Fatal(err)
	}
	if _, err := a.Show("deploy/notes"); err != nil {
		t.Fatal(err)
	}
	st := store.Open(dir)
	if st.Tab() != "deploy" {
		t.Errorf("the root sticky.json should name the tab: %q", st.Tab())
	}
	if v, _ := st.Views(); v["deploy/notes.md"].Open == nil {
		t.Errorf("the note is opened in the tab's own file: %+v", v)
	}
	if _, err := a.Show("plan"); err != nil || st.Tab() != "" {
		t.Errorf("showing a root note switches back: %v, %q", err, st.Tab())
	}
}

// The README promises `stickypane show README.md`: a file of the project
// named without a folder is linked and shown, as long as no note has that
// name. A note of that name wins.
func TestShowABareFileNameOfTheProject(t *testing.T) {
	a, dir := newAPI(t, map[string]string{"plan.md": "the plan\n"})
	root := filepath.Dir(dir)
	for name, body := range map[string]string{"README.md": "# Readme\n", "plan": "not the note\n"} {
		if err := os.WriteFile(filepath.Join(root, name), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	t.Chdir(root)
	got, err := a.Show("README.md")
	if err != nil {
		if errors.Is(err, os.ErrPermission) {
			t.Skip("symlinks are not available:", err)
		}
		t.Fatalf("Show(README.md) = %v", err)
	}
	if got != (Shown{Note: "README.md", Open: true, Linked: "../README.md"}) || !has(dir, "README.md") {
		t.Errorf("Show(README.md) = %+v", got)
	}
	if got, err := a.Show("plan"); err != nil || got != (Shown{Note: "plan.md", Open: true}) {
		t.Errorf("a note wins over a file of the same name: %+v, %v", got, err)
	}
	if _, err := a.Show("missing.md"); err == nil {
		t.Error("a name that is neither a note nor a file cannot be shown")
	}
}
