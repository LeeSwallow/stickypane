package index

import (
	"io/fs"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// golden is the home stickypane's own tests write and pin
// (internal/adapter/registry/testdata/home in the main module): one board,
// /work/demo, set up and opened in the terminal (pid 4242, running) and
// served by a bridge (pid 4343) that has ended.
const golden = "../../../internal/adapter/registry/testdata/home"

// copyHome copies the golden home into a temporary folder, for the index
// makes its database in the home it opens.
func copyHome(t *testing.T) string {
	t.Helper()
	home := filepath.Join(t.TempDir(), ".stickypane")
	err := filepath.WalkDir(golden, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(golden, p)
		if d.IsDir() {
			return os.MkdirAll(filepath.Join(home, rel), 0o755)
		}
		b, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		return os.WriteFile(filepath.Join(home, rel), b, 0o644)
	})
	if err != nil {
		t.Fatal(err)
	}
	return home
}

// pids is a ProcessChecker that knows which pids live.
type pids map[int]bool

func (p pids) Alive(pid int, _ time.Time) bool { return p[pid] }

func TestIndexesTheGoldenHome(t *testing.T) {
	home := copyHome(t)
	ix := openIdx(t, home)
	ix.Procs = pids{4242: true}
	if err := ix.Ingest(); err != nil {
		t.Fatal(err)
	}
	if ix.Skipped() != 0 {
		t.Fatalf("skipped %d golden lines", ix.Skipped())
	}
	ps := projects(t, ix)
	if len(ps) != 1 || ps[0].Path != "/work/demo" || ps[0].Title != "demo" {
		t.Fatalf("projects = %+v", ps)
	}
	if n := dayEvents(t, ix, "/work/demo"); n != 5 {
		t.Fatalf("day events = %d", n)
	}
	type sess struct {
		Pid   int
		Kind  string
		Ended *string
	}
	var ss []sess
	ix.store.DB().Table("session").Select("pid, kind, ended_at AS ended").Order("pid").Scan(&ss)
	if len(ss) != 2 || ss[0].Pid != 4242 || ss[0].Kind != "tui" || ss[0].Ended != nil || ss[1].Pid != 4343 || ss[1].Ended == nil {
		t.Fatalf("sessions = %+v", ss)
	}
	g, err := ix.Sessions(time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if len(g.Active) != 1 || len(g.Active[0].Live) != 1 || g.Active[0].Live[0].PID != 4242 || g.Active[0].Live[0].Kind != "tui" {
		t.Fatalf("groups = %+v", g)
	}

	// The terminal's process ends without a word: its session ends, and
	// every record is still there, years later, for stickypane to prune.
	ix.Procs = pids{}
	ix.Ingest()
	ix.Prune(time.Now().AddDate(2, 0, 0))
	for _, f := range []string{"events/2026-10-28.jsonl", "run/4242.json"} {
		if _, err := os.Stat(filepath.Join(home, f)); err != nil {
			t.Errorf("%s: %v", f, err)
		}
	}
}
