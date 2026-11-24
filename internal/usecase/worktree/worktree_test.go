package worktree

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/LeeSwallow/stickypane/internal/adapter/registry"
	"github.com/LeeSwallow/stickypane/internal/adapter/store"
	"github.com/LeeSwallow/stickypane/internal/core/doc"
)

var fixed = time.Date(2026, 11, 24, 9, 30, 0, 0, time.FixedZone("KST", 9*60*60))

// fixture is a repository with a board in its main worktree, and a home
// where records are kept.
type fixture struct {
	t          *testing.T
	dir, main  string
	board      Board
	home       string
	recordsOff bool
}

func newFixture(t *testing.T, records bool) *fixture {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("no git")
	}
	dir := t.TempDir()
	dir, _ = filepath.EvalSymlinks(dir)
	r := &fixture{t: t, dir: dir, main: filepath.Join(dir, "main"), home: filepath.Join(dir, "home")}
	os.MkdirAll(filepath.Join(r.main, store.DirName), 0o755)
	os.MkdirAll(r.home, 0o755)
	if records {
		os.WriteFile(filepath.Join(r.home, registry.AppFile), []byte("{}\n"), 0o644)
	}
	r.git("init", "-q", "-b", "main")
	r.git("commit", "-q", "--allow-empty", "-m", "first")
	r.board = Board{
		Store:   store.Open(filepath.Join(r.main, store.DirName)),
		Records: registry.Writer{Home: r.home, Now: func() time.Time { return fixed }, PID: 1},
		Now:     func() time.Time { return fixed },
	}
	r.board.Store.NewFolder = r.board.NewFolder
	return r
}

func (r *fixture) git(args ...string) {
	r.t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = r.main
	cmd.Env = append(os.Environ(), "GIT_AUTHOR_NAME=t", "GIT_AUTHOR_EMAIL=t@t", "GIT_COMMITTER_NAME=t", "GIT_COMMITTER_EMAIL=t@t")
	if out, err := cmd.CombinedOutput(); err != nil {
		r.t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}

func (r *fixture) sync() []Worktree {
	r.t.Helper()
	list, err := r.board.Sync()
	if err != nil {
		r.t.Fatal(err)
	}
	return list
}

func (r *fixture) find(list []Worktree, name string) Worktree {
	r.t.Helper()
	for _, w := range list {
		if w.Name == name {
			return w
		}
	}
	r.t.Fatalf("no %s in %+v", name, list)
	return Worktree{}
}

func (r *fixture) overview(name string) doc.Document {
	r.t.Helper()
	b, err := os.ReadFile(filepath.Join(r.board.Store.Dir, Tab, name, Overview))
	if err != nil {
		r.t.Fatal(err)
	}
	return doc.Parse(b)
}

func (r *fixture) events() string {
	b, _ := os.ReadFile(filepath.Join(r.home, "events", "2026-11-24.jsonl"))
	return string(b)
}

func TestAWorktreeIsRecordedButGetsNoFolderUntilWritten(t *testing.T) {
	r := newFixture(t, true)
	wt := filepath.Join(r.dir, "wt", "feat-a")
	r.git("worktree", "add", "-q", "-b", "feat/a", wt)

	w := r.find(r.sync(), "feat-a")
	if w.State != registry.StateOpen || w.Path != wt || w.Branch != "feat/a" || w.Folder {
		t.Fatalf("worktree = %+v", w)
	}
	if _, err := os.Stat(filepath.Join(r.board.Store.Dir, Tab)); err == nil {
		t.Fatal("a folder was made at sync")
	}
	if !strings.Contains(r.events(), `"type":"worktree.open"`) {
		t.Fatalf("events = %s", r.events())
	}
	// Synced again, nothing new is recorded.
	before := r.events()
	r.sync()
	if r.events() != before {
		t.Fatalf("a second sync recorded %s", strings.TrimPrefix(r.events(), before))
	}

	// The first write makes the folder and its overview.
	if err := r.board.Store.Write(Tab+"/feat-a/notes.md", []byte("hi\n")); err != nil {
		t.Fatal(err)
	}
	d := r.overview("feat-a")
	if s, _ := d.Get("state"); s != "open" {
		t.Fatalf("state = %q", s)
	}
	if b, _ := d.Get("branch"); b != "feat/a" {
		t.Fatalf("branch = %q", b)
	}
	if strings.Contains(string(d.Bytes()), wt) {
		t.Fatal("the overview carries the path")
	}
	if !r.find(r.sync(), "feat-a").Folder {
		t.Fatal("the folder is not seen")
	}
	// A write to a folder that is no worktree gets no overview.
	r.board.Store.Write(Tab+"/mine/notes.md", []byte("hi\n"))
	if _, err := os.Stat(filepath.Join(r.board.Store.Dir, Tab, "mine", Overview)); err == nil {
		t.Fatal("a person's folder got an overview")
	}
}

func TestAMovedWorktreeKeepsItsNameAndAClosedOneItsFolder(t *testing.T) {
	r := newFixture(t, true)
	wt := filepath.Join(r.dir, "wt", "feat-a")
	r.git("worktree", "add", "-q", "-b", "feat/a", wt)
	r.sync()
	r.board.Store.Write(Tab+"/feat-a/notes.md", []byte("hi\n"))

	moved := filepath.Join(r.dir, "elsewhere")
	r.git("worktree", "move", wt, moved)
	w := r.find(r.sync(), "feat-a")
	if w.Path != moved || w.State != registry.StateOpen {
		t.Fatalf("after the move = %+v", w)
	}
	if !strings.Contains(r.events(), `"type":"worktree.move"`) || !strings.Contains(r.events(), `"from":"`+wt+`"`) {
		t.Fatalf("events = %s", r.events())
	}
	recs := r.board.Records.Worktrees(filepath.Join(r.main, ".git"))
	if len(recs) != 1 || recs[0].Path != moved {
		t.Fatalf("record = %+v", recs)
	}

	r.git("worktree", "remove", moved)
	w = r.find(r.sync(), "feat-a")
	if w.State != registry.StateClosed || !w.Folder {
		t.Fatalf("after the removal = %+v", w)
	}
	d := r.overview("feat-a")
	if s, _ := d.Get("state"); s != "closed" {
		t.Fatalf("state = %q", s)
	}
	if c, _ := d.Get("closed"); c != "2026-11-24 09:30" {
		t.Fatalf("closed = %q", c)
	}
	if _, err := os.Stat(filepath.Join(r.board.Store.Dir, Tab, "feat-a", "notes.md")); err != nil {
		t.Fatal("the folder's notes are gone")
	}
	if !strings.Contains(r.events(), `"type":"worktree.close"`) {
		t.Fatalf("events = %s", r.events())
	}

	// The branch comes back: the folder opens again.
	r.git("worktree", "add", "-q", filepath.Join(r.dir, "again"), "feat/a")
	w = r.find(r.sync(), "feat-a")
	d = r.overview("feat-a")
	if s, _ := d.Get("state"); s != "open" || w.State != registry.StateOpen {
		t.Fatalf("reopened: state %q, %+v", s, w)
	}
	if _, ok := d.Get("closed"); ok {
		t.Fatal("closed is still set")
	}
}

// Without records, the folder's overview is what is known: a removed
// worktree still closes its folder, and a folder a person made is left.
func TestWithoutRecordsTheFoldersCarryTheState(t *testing.T) {
	r := newFixture(t, false)
	wt := filepath.Join(r.dir, "wt", "feat-a")
	r.git("worktree", "add", "-q", "-b", "feat/a", wt)
	r.sync()
	r.board.Store.Write(Tab+"/feat-a/notes.md", []byte("hi\n"))
	os.MkdirAll(filepath.Join(r.board.Store.Dir, Tab, "person"), 0o755)
	person := "---\ntitle: mine\n---\nmine\n"
	os.WriteFile(filepath.Join(r.board.Store.Dir, Tab, "person", Overview), []byte(person), 0o644)

	// Moved to a folder of another name, it is found by its branch.
	r.git("worktree", "move", wt, filepath.Join(r.dir, "renamed"))
	if w := r.find(r.sync(), "feat-a"); w.State != registry.StateOpen {
		t.Fatalf("after the move = %+v", w)
	}
	r.git("worktree", "remove", filepath.Join(r.dir, "renamed"))
	r.sync()
	if s, _ := r.overview("feat-a").Get("state"); s != "closed" {
		t.Fatalf("state = %q", s)
	}
	if b, _ := os.ReadFile(filepath.Join(r.board.Store.Dir, Tab, "person", Overview)); string(b) != person {
		t.Fatalf("a person's note changed: %q", b)
	}
	if ents, _ := os.ReadDir(r.home); len(ents) != 0 {
		t.Fatal("records were written")
	}
}

func TestNamesLockedAndWaiting(t *testing.T) {
	r := newFixture(t, true)
	// Claude Code's way: branch worktree-chart in .claude/worktrees/chart.
	claude := filepath.Join(r.main, ".claude", "worktrees", "chart")
	r.git("worktree", "add", "-q", "-b", "worktree-chart", claude)
	r.git("worktree", "lock", claude)
	other := filepath.Join(r.dir, "other", "chart")
	r.git("worktree", "add", "-q", "-b", "chart2", other)
	r.git("worktree", "add", "-q", "--detach", filepath.Join(r.dir, "loose"))

	list := r.sync()
	c := r.find(list, "chart")
	if c.Branch != "worktree-chart" || !c.Running {
		t.Fatalf("chart = %+v", c)
	}
	var hashed Worktree
	for _, w := range list {
		if strings.HasPrefix(w.Name, "chart-") {
			hashed = w
		}
	}
	if hashed.Path != other || len(hashed.Name) != len("chart-")+4 {
		t.Fatalf("the second chart = %+v in %+v", hashed, list)
	}
	if l := r.find(list, "loose"); l.Branch != "" || l.State != registry.StateOpen {
		t.Fatalf("detached = %+v", l)
	}
	// Names hold from one sync to the next.
	if again := r.sync(); r.find(again, hashed.Name).Path != other {
		t.Fatalf("names moved: %+v", again)
	}

	r.board.Store.Write(Tab+"/chart/ask.md", []byte("---\ntype: form\n---\nok?\n"))
	r.board.Store.Write(Tab+"/chart/done.md", []byte("---\ntype: form\nsubmitted: OK\n---\nok?\n"))
	if w := r.find(r.sync(), "chart"); w.Waiting != 1 {
		t.Fatalf("waiting = %d", w.Waiting)
	}
}

// Two writers making the same folder at once end with one overview.
func TestTwoFirstWritesMakeOneFolder(t *testing.T) {
	r := newFixture(t, true)
	r.git("worktree", "add", "-q", "-b", "feat/a", filepath.Join(r.dir, "feat-a"))
	r.sync()
	var wg sync.WaitGroup
	for i := range 4 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			st := store.Open(r.board.Store.Dir)
			st.NewFolder = r.board.NewFolder
			st.Write(Tab+"/feat-a/n"+string(rune('0'+i))+".md", []byte("x\n"))
		}()
	}
	wg.Wait()
	ents, _ := os.ReadDir(filepath.Join(r.board.Store.Dir, Tab, "feat-a"))
	if len(ents) != 5 {
		t.Fatalf("folder = %v", ents)
	}
}

func TestOutsideARepositoryNothingHappens(t *testing.T) {
	dir := filepath.Join(t.TempDir(), store.DirName)
	os.Mkdir(dir, 0o755)
	b := Board{Store: store.Open(dir), Now: func() time.Time { return fixed }}
	if list, err := b.Sync(); list != nil || err != nil {
		t.Fatalf("Sync = %v, %v", list, err)
	}
	if b.NewFolder(Tab+"/x") != nil {
		t.Fatal("NewFolder outside a repository")
	}
}

// Without records, a detached worktree finds its folder by the folder's
// name, the last element of its path: it stays open and gets no new name.
func TestWithoutRecordsADetachedWorktreeKeepsItsFolder(t *testing.T) {
	r := newFixture(t, false)
	r.git("worktree", "add", "-q", "--detach", filepath.Join(r.dir, "loose"))
	r.sync()
	if err := r.board.Store.Write(Tab+"/loose/plan.md", []byte("hi\n")); err != nil {
		t.Fatal(err)
	}
	list := r.sync()
	if len(list) != 1 || list[0].Name != "loose" || list[0].State != registry.StateOpen {
		t.Fatalf("list = %+v", list)
	}
	if s, _ := r.overview("loose").Get("state"); s != "open" {
		t.Fatalf("state = %q", s)
	}
}
