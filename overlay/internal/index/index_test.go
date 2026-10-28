package index

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// alive is a ProcessChecker that answers the same for every process.
type alive bool

func (a alive) Alive(int, time.Time) bool { return bool(a) }

func newHome(t *testing.T) string {
	t.Helper()
	h := filepath.Join(t.TempDir(), ".stickypane")
	if err := os.MkdirAll(filepath.Join(h, "events"), 0o755); err != nil {
		t.Fatal(err)
	}
	return h
}

func openIdx(t *testing.T, home string) *Index {
	t.Helper()
	ix, err := Open(home)
	if err != nil {
		t.Fatal(err)
	}
	ix.Procs = alive(false)
	t.Cleanup(func() { ix.Close() })
	return ix
}

// board makes a project folder with a .sticky/ in it.
func board(t *testing.T, name string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), name)
	if err := os.MkdirAll(filepath.Join(p, ".sticky"), 0o755); err != nil {
		t.Fatal(err)
	}
	return p
}

// oldBoard is a board whose .sticky/ has not changed for two days.
func oldBoard(t *testing.T, name string) string {
	t.Helper()
	p := board(t, name)
	old := time.Now().Add(-48 * time.Hour)
	if err := os.Chtimes(filepath.Join(p, ".sticky"), old, old); err != nil {
		t.Fatal(err)
	}
	return p
}

func ev(at time.Time, typ, project string, extra string) string {
	return fmt.Sprintf(`{"v":1,"at":%q,"type":%q,"project":%q%s}`, at.Format(time.RFC3339), typ, project, extra)
}

func writeEvents(t *testing.T, home, date string, lines ...string) string {
	t.Helper()
	f := filepath.Join(home, "events", date+".jsonl")
	if err := os.WriteFile(f, []byte(strings.Join(lines, "\n")+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	return f
}

func appendRaw(t *testing.T, f, s string) {
	t.Helper()
	fh, err := os.OpenFile(f, os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		t.Fatal(err)
	}
	defer fh.Close()
	fh.WriteString(s)
}

func projects(t *testing.T, ix *Index) []Project {
	t.Helper()
	ps, err := ix.Projects()
	if err != nil {
		t.Fatal(err)
	}
	return ps
}

func dayEvents(t *testing.T, ix *Index, path string) int {
	t.Helper()
	var n int
	err := ix.store.DB().Table("day").Select("COALESCE(SUM(events),0)").
		Joins("JOIN project ON project.id = day.project_id").Where("project.path = ?", path).Row().Scan(&n)
	if err != nil {
		t.Fatal(err)
	}
	return n
}

func date(t time.Time) string { return t.Format("2006-01-02") }

func TestOpenSetsSchemaVersionAndWAL(t *testing.T) {
	ix := openIdx(t, newHome(t))
	var v, mode string
	ix.store.DB().Table("meta").Select("value").Where("key = ?", "schema_version").Row().Scan(&v)
	ix.store.DB().Raw(`PRAGMA journal_mode`).Row().Scan(&mode)
	if v != "1" || mode != "wal" {
		t.Fatalf("schema_version=%q journal_mode=%q", v, mode)
	}
}

func TestIngestReadsOnlyNewLines(t *testing.T) {
	home, p := newHome(t), board(t, "a")
	now := time.Now()
	f := writeEvents(t, home, date(now), ev(now, "project.touch", p, `,"cmd":"card"`))
	ix := openIdx(t, home)
	if err := ix.Ingest(); err != nil {
		t.Fatal(err)
	}
	if err := ix.Ingest(); err != nil { // nothing new: no double count
		t.Fatal(err)
	}
	if n := dayEvents(t, ix, p); n != 1 {
		t.Fatalf("after two ingests of one line: %d events", n)
	}
	appendRaw(t, f, ev(now, "project.touch", p, `,"cmd":"edit"`)+"\n")
	ix.Ingest()
	if n := dayEvents(t, ix, p); n != 2 {
		t.Fatalf("after one new line: %d events", n)
	}
}

func TestIngestWaitsForAPartialLastLine(t *testing.T) {
	home, p := newHome(t), board(t, "a")
	now := time.Now()
	f := writeEvents(t, home, date(now), ev(now, "project.touch", p, ""))
	whole := ev(now, "project.touch", p, `,"cmd":"x"`)
	appendRaw(t, f, whole[:len(whole)-10])
	ix := openIdx(t, home)
	ix.Ingest()
	if n := dayEvents(t, ix, p); n != 1 {
		t.Fatalf("partial line counted: %d", n)
	}
	appendRaw(t, f, whole[len(whole)-10:]+"\n")
	ix.Ingest()
	if n := dayEvents(t, ix, p); n != 2 {
		t.Fatalf("completed line not read: %d", n)
	}
}

func TestIngestSkipsBrokenLines(t *testing.T) {
	home, p := newHome(t), board(t, "a")
	now := time.Now()
	writeEvents(t, home, date(now), ev(now, "project.touch", p, ""), `{not json`, `{"v":1,"type":"project.touch"}`, ev(now, "project.touch", p, ""))
	ix := openIdx(t, home)
	if err := ix.Ingest(); err != nil {
		t.Fatal(err)
	}
	if n := dayEvents(t, ix, p); n != 2 {
		t.Fatalf("events = %d", n)
	}
	if s := ix.Skipped(); s != 2 {
		t.Fatalf("skipped = %d", s)
	}
}

func TestPruneMarksAMissingFolderAndRestoresItWhenBack(t *testing.T) {
	home, p := newHome(t), board(t, "a")
	now := time.Now()
	writeEvents(t, home, date(now), ev(now, "project.init", p, ""))
	ix := openIdx(t, home)
	ix.Ingest()
	os.Rename(filepath.Join(p, ".sticky"), filepath.Join(p, ".gone"))
	removed, err := ix.Prune(now)
	if err != nil || len(removed) != 1 || removed[0].Path != p {
		t.Fatalf("removed=%v err=%v", removed, err)
	}
	g, _ := ix.Sessions(now)
	if len(g.Active)+len(g.Days) != 0 || len(g.Removed) != 1 {
		t.Fatalf("groups = %+v", g)
	}
	os.Rename(filepath.Join(p, ".gone"), filepath.Join(p, ".sticky"))
	if removed, _ = ix.Prune(now); len(removed) != 0 {
		t.Fatalf("removed again: %v", removed)
	}
	if ps := projects(t, ix); len(ps) != 1 || ps[0].RemovedAt != nil {
		t.Fatalf("not restored: %+v", ps)
	}
}

func TestPruneDropsAFolderRemovedWhileOpen(t *testing.T) {
	home, p := newHome(t), oldBoard(t, "a")
	now := time.Now()
	writeEvents(t, home, date(now), ev(now, "project.touch", p, ""))
	ix := openIdx(t, home)
	ix.Ingest()
	if removed, _ := ix.Prune(now); len(removed) != 0 {
		t.Fatalf("present folder removed: %v", removed)
	}
	if g, _ := ix.Sessions(now); len(g.Days) != 1 {
		t.Fatalf("not listed: %+v", g)
	}
	os.RemoveAll(p)
	removed, _ := ix.Prune(now)
	if len(removed) != 1 {
		t.Fatalf("removed = %v", removed)
	}
	if g, _ := ix.Sessions(now); len(g.Days) != 0 || len(g.Removed) != 1 {
		t.Fatalf("still listed: %+v", g)
	}
	// An event for the returned folder clears the mark on the next ingest.
	os.MkdirAll(filepath.Join(p, ".sticky"), 0o755)
	appendRaw(t, filepath.Join(home, "events", date(now)+".jsonl"), ev(now, "project.touch", p, "")+"\n")
	ix.Ingest()
	if ps := projects(t, ix); ps[0].RemovedAt != nil {
		t.Fatal("event did not restore")
	}
}

func TestRestoreClearsRemovedAt(t *testing.T) {
	home, p := newHome(t), board(t, "a")
	now := time.Now()
	writeEvents(t, home, date(now), ev(now, "project.touch", p, ""))
	ix := openIdx(t, home)
	ix.Ingest()
	os.RemoveAll(p)
	removed, _ := ix.Prune(now)
	if err := ix.Restore(removed[0].ID); err != nil {
		t.Fatal(err)
	}
	if ps := projects(t, ix); ps[0].RemovedAt != nil {
		t.Fatal("still removed")
	}
}

func TestPruneDeletesRowsAfter30Days(t *testing.T) {
	home, p := newHome(t), board(t, "a")
	now := time.Now()
	writeEvents(t, home, date(now), ev(now, "project.touch", p, ""))
	ix := openIdx(t, home)
	ix.Ingest()
	os.RemoveAll(p)
	ix.Prune(now)
	ix.Prune(now.Add(29 * 24 * time.Hour))
	if len(projects(t, ix)) != 1 {
		t.Fatal("deleted before 30 days")
	}
	ix.Prune(now.Add(31 * 24 * time.Hour))
	if len(projects(t, ix)) != 0 {
		t.Fatal("kept after 30 days")
	}
	if n := dayEvents(t, ix, p); n != 0 {
		t.Fatal("day rows not cascaded")
	}
}

func TestDeadPidsEndTheirSessions(t *testing.T) {
	home, p := newHome(t), board(t, "a")
	now := time.Now()
	writeEvents(t, home, date(now), ev(now, "session.start", p, `,"pid":4242,"kind":"tui"`))
	run := filepath.Join(home, "run")
	os.MkdirAll(run, 0o755)
	body := fmt.Sprintf(`{"pid":4242,"project":%q,"kind":"tui","started":%q}`, p, now.Format(time.RFC3339))
	os.WriteFile(filepath.Join(run, "4242.json"), []byte(body), 0o644)
	os.WriteFile(filepath.Join(run, "touch-abc"), nil, 0o644)
	os.WriteFile(filepath.Join(run, ".tmp-1"), []byte("junk"), 0o644)
	ix := openIdx(t, home)

	ix.Procs = alive(true)
	ix.Ingest()
	if g, _ := ix.Sessions(now); len(g.Active) != 1 || len(g.Active[0].Live) != 1 {
		t.Fatalf("live session missing: %+v", g)
	}
	if _, err := os.Stat(filepath.Join(run, "4242.json")); err != nil {
		t.Fatal("live run file removed")
	}

	// Alive says no: process gone, or the pid belongs to another program.
	// Its session ends; its file stays, for stickypane to remove.
	ix.Procs = alive(false)
	ix.Ingest()
	if g, _ := ix.Sessions(now); len(g.Active) != 1 || len(g.Active[0].Live) != 0 {
		t.Fatalf("dead session listed live: %+v", g)
	}
	for _, f := range []string{"4242.json", "touch-abc", ".tmp-1"} {
		if _, err := os.Stat(filepath.Join(run, f)); err != nil {
			t.Fatalf("%s was touched", f)
		}
	}
	var ended *string
	ix.store.DB().Table("session").Select("ended_at").Where("pid = ?", 4242).Row().Scan(&ended)
	if ended == nil {
		t.Fatal("ended_at not filled")
	}
}

func TestRebuildFromEventsWhenTheDBIsCorrupt(t *testing.T) {
	home, p := newHome(t), board(t, "a")
	now := time.Now()
	writeEvents(t, home, date(now), ev(now, "project.touch", p, ""))
	ix := openIdx(t, home)
	ix.Ingest()
	ix.Close()
	os.Remove(filepath.Join(home, "stickypane.db-wal"))
	os.Remove(filepath.Join(home, "stickypane.db-shm"))
	if err := os.WriteFile(filepath.Join(home, "stickypane.db"), []byte(strings.Repeat("not a database ", 500)), 0o644); err != nil {
		t.Fatal(err)
	}
	ix2 := openIdx(t, home)
	if n := dayEvents(t, ix2, p); n != 1 {
		t.Fatalf("not rebuilt: %d", n)
	}
	broken, _ := filepath.Glob(filepath.Join(home, "stickypane.db.broken-*"))
	if len(broken) != 1 {
		t.Fatalf("broken copies: %v", broken)
	}
}

func TestDeletedEventFileForgetsItsOffset(t *testing.T) {
	home, p := newHome(t), board(t, "a")
	now := time.Now()
	f := writeEvents(t, home, date(now), ev(now, "project.touch", p, ""))
	ix := openIdx(t, home)
	ix.Ingest()
	os.Remove(f)
	ix.Ingest()
	var n int
	ix.store.DB().Table("ingest").Select("COUNT(*)").Row().Scan(&n)
	if n != 0 {
		t.Fatalf("ingest rows = %d", n)
	}
	if got := dayEvents(t, ix, p); got != 1 {
		t.Fatalf("already-applied record lost: %d", got)
	}
	// A new file of the same name is read from its start.
	writeEvents(t, home, date(now), ev(now, "project.touch", p, ""))
	ix.Ingest()
	if got := dayEvents(t, ix, p); got != 2 {
		t.Fatalf("new file not read: %d", got)
	}
}

// The app deletes nothing in ~/.stickypane: stickypane prunes its own old
// events and dead run files (internal/registry Prune). Changed in #24: the
// app used to remove events older than 90 days and dead run files itself.
func TestPruneLeavesTheRecordsToStickypane(t *testing.T) {
	home, p := newHome(t), board(t, "a")
	now := time.Now()
	old, fresh := now.AddDate(0, 0, -91), now.AddDate(0, 0, -89)
	fo := writeEvents(t, home, date(old), ev(old, "project.touch", p, ""))
	ff := writeEvents(t, home, date(fresh), ev(fresh, "project.touch", p, ""))
	run := filepath.Join(home, "run")
	os.MkdirAll(run, 0o755)
	dead := filepath.Join(run, "4242.json")
	os.WriteFile(dead, []byte(fmt.Sprintf(`{"kind":"tui","pid":4242,"project":%q,"started":%q}`, p, old.Format(time.RFC3339))), 0o644)
	ix := openIdx(t, home)
	ix.Ingest()
	ix.Prune(now.AddDate(1, 0, 0))
	for _, f := range []string{fo, ff, dead} {
		if _, err := os.Stat(f); err != nil {
			t.Fatalf("%s removed by the app", f)
		}
	}
	if n := dayEvents(t, ix, p); n != 2 {
		t.Fatalf("day table lost rows: %d", n)
	}
}

func TestSessionsGroupsByLocalDay(t *testing.T) {
	home := newHome(t)
	loc := time.Local
	now := time.Now()
	a, b, c := oldBoard(t, "a"), oldBoard(t, "b"), board(t, "c")
	// Two days with a local midnight between them, minutes apart.
	midnight := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc).AddDate(0, 0, -3)
	today := midnight.Add(30 * time.Minute)
	yest := midnight.Add(-30 * time.Minute)
	// c is active: its .sticky/ changed within 30 minutes of now. Its board
	// names it Sea (board.load, through the shell).
	os.WriteFile(filepath.Join(c, ".sticky", "sticky.json"), []byte(`{}`), 0o644)
	writeEvents(t, home, date(today), ev(today, "project.touch", a, ""), ev(today, "project.touch", c, ""))
	writeEvents(t, home, date(yest), ev(yest, "project.touch", b, ""))
	ix := openIdx(t, home)
	ix.Ingest()
	if err := ix.Retitle(c, "Sea"); err != nil {
		t.Fatal(err)
	}
	g, err := ix.Sessions(now)
	if err != nil {
		t.Fatal(err)
	}
	if len(g.Active) != 1 || g.Active[0].Title != "Sea" {
		t.Fatalf("active = %+v", g.Active)
	}
	if len(g.Days) != 2 || g.Days[0].Date != date(today) || g.Days[1].Date != date(yest) {
		t.Fatalf("days = %+v", g.Days)
	}
	if len(g.Days[0].Rows) != 1 || g.Days[0].Rows[0].Path != a || g.Days[1].Rows[0].Path != b || g.Days[1].Rows[0].Title != "b" {
		t.Fatalf("rows = %+v", g.Days)
	}
	// Thirty-one minutes later c is no longer active.
	g, _ = ix.Sessions(now.Add(31 * time.Minute))
	if len(g.Active) != 0 || len(g.Days) != 2 || len(g.Days[0].Rows) != 2 {
		t.Fatalf("after 31 minutes: %+v", g)
	}
}

func TestConcurrentOpenOfAFreshHomeKeepsEveryone(t *testing.T) {
	home := newHome(t)
	const n = 4
	errs := make(chan error, n)
	ixs := make(chan *Index, n)
	for i := 0; i < n; i++ {
		go func() {
			ix, err := Open(home)
			errs <- err
			ixs <- ix
		}()
	}
	for i := 0; i < n; i++ {
		if err := <-errs; err != nil {
			t.Error(err)
		}
		if ix := <-ixs; ix != nil {
			ix.Close()
		}
	}
	if broken, _ := filepath.Glob(filepath.Join(home, "stickypane.db.broken-*")); len(broken) != 0 {
		t.Fatalf("a healthy database was moved aside: %v", broken)
	}
}

func TestPermissionErrorIsNotCorruption(t *testing.T) {
	home := newHome(t)
	ix := openIdx(t, home)
	ix.Close()
	db := filepath.Join(home, "stickypane.db")
	if err := os.Chmod(db, 0); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chmod(db, 0o644) })
	if f, err := os.Open(db); err == nil {
		f.Close()
		t.Skip("file stays readable (running as root?)")
	}
	if _, err := Open(home); err == nil {
		t.Fatal("expected an error")
	}
	if broken, _ := filepath.Glob(filepath.Join(home, "stickypane.db.broken-*")); len(broken) != 0 {
		t.Fatalf("renamed on a permission error: %v", broken)
	}
	if _, err := os.Stat(db); err != nil {
		t.Fatal("database gone")
	}
}

func TestConcurrentIngestAppliesEachLineOnce(t *testing.T) {
	home, p := newHome(t), board(t, "a")
	now := time.Now()
	var lines []string
	for i := 0; i < 200; i++ {
		lines = append(lines, ev(now, "project.touch", p, ""))
	}
	writeEvents(t, home, date(now), lines...)
	a, b := openIdx(t, home), openIdx(t, home)
	errs := make(chan error, 2)
	for _, ix := range []*Index{a, b} {
		go func() { errs <- ix.Ingest() }()
	}
	for i := 0; i < 2; i++ {
		if err := <-errs; err != nil {
			t.Fatal(err)
		}
	}
	if n := dayEvents(t, a, p); n != 200 {
		t.Fatalf("day.events = %d, want 200", n)
	}
}

// The app does not read sticky.json: a project is named after its folder
// until its board, loaded through the bridge, names it. Changed in #24:
// the index used to read the title from .sticky/sticky.json itself.
func TestTitleIsTheFolderUntilTheBoardNamesIt(t *testing.T) {
	home, p := newHome(t), oldBoard(t, "folder")
	now := time.Now()
	writeEvents(t, home, date(now), ev(now, "project.touch", p, ""))
	os.WriteFile(filepath.Join(p, ".sticky", "sticky.json"), []byte(`{"title":"Ignored"}`), 0o644)
	ix := openIdx(t, home)
	ix.Ingest()
	if ps := projects(t, ix); ps[0].Title != "folder" {
		t.Fatalf("title = %q", ps[0].Title)
	}
	if err := ix.Retitle(p+"/", "Named"); err != nil {
		t.Fatal(err)
	}
	ix.Retitle(p, "") // an empty title changes nothing
	ix.Retitle("/elsewhere", "Other")
	ix.Prune(now)
	if ps := projects(t, ix); len(ps) != 1 || ps[0].Title != "Named" {
		t.Fatalf("projects = %+v", ps)
	}
}
