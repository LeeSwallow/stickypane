package gormstore

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"
	"time"

	_ "github.com/glebarez/go-sqlite" // the driver, to write a database as the first version did

	"github.com/LeeSwallow/stickypane/overlay/internal/index/model"
	"github.com/LeeSwallow/stickypane/overlay/internal/index/ports"
)

// v1 is the schema the first version of the index made with hand-written
// SQL. A database made by it is opened as it is.
const v1 = `
CREATE TABLE project (id INTEGER PRIMARY KEY, path TEXT NOT NULL UNIQUE, title TEXT,
  first_seen TEXT NOT NULL, last_seen TEXT NOT NULL, removed_at TEXT);
CREATE TABLE day (project_id INTEGER NOT NULL REFERENCES project(id) ON DELETE CASCADE,
  date TEXT NOT NULL, events INTEGER NOT NULL DEFAULT 0, first_at TEXT NOT NULL, last_at TEXT NOT NULL,
  PRIMARY KEY (project_id, date));
CREATE TABLE session (pid INTEGER NOT NULL, project_id INTEGER NOT NULL REFERENCES project(id) ON DELETE CASCADE,
  kind TEXT NOT NULL, started_at TEXT NOT NULL, ended_at TEXT, PRIMARY KEY (pid, started_at));
CREATE TABLE ingest (file TEXT PRIMARY KEY, offset INTEGER NOT NULL);
CREATE TABLE meta (key TEXT PRIMARY KEY, value TEXT);
INSERT INTO meta VALUES ('schema_version', '1');
INSERT INTO project VALUES (1, '/w/a', NULL, '2026-10-28T14:02:11+09:00', '2026-10-28T15:00:00+09:00', NULL);
INSERT INTO day VALUES (1, '2026-10-28', 3, '2026-10-28T14:02:11+09:00', '2026-10-28T15:00:00+09:00');
`

func write(t *testing.T, path, script string) {
	t.Helper()
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err := db.Exec(script); err != nil {
		t.Fatal(err)
	}
}

func TestOpensADatabaseOfTheFirstVersion(t *testing.T) {
	path := filepath.Join(t.TempDir(), "stickypane.db")
	write(t, path, v1)
	s, rebuilt, err := Open(path)
	if err != nil || rebuilt {
		t.Fatalf("rebuilt=%v err=%v", rebuilt, err)
	}
	defer s.Close()
	ps, err := s.Repos().Projects.All()
	if err != nil || len(ps) != 1 || ps[0].Path != "/w/a" || ps[0].Title != "" || ps[0].LastSeen.Hour() != 15 {
		t.Fatalf("projects = %+v, %v", ps, err)
	}
	d, ok, err := s.Repos().Days.Get(1, "2026-10-28")
	if !ok || err != nil || d.Events != 3 {
		t.Fatalf("day = %+v %v %v", d, ok, err)
	}
	// Deleting the project takes its days: the foreign keys are on.
	if err := s.Repos().Projects.Delete(1); err != nil {
		t.Fatal(err)
	}
	if _, ok, _ := s.Repos().Days.Get(1, "2026-10-28"); ok {
		t.Fatal("day kept after its project")
	}
}

func TestALaterSchemaVersionIsSetAside(t *testing.T) {
	path := filepath.Join(t.TempDir(), "stickypane.db")
	write(t, path, `CREATE TABLE meta (key TEXT PRIMARY KEY, value TEXT); INSERT INTO meta VALUES ('schema_version', '99');`)
	s, rebuilt, err := Open(path)
	if err != nil || !rebuilt {
		t.Fatalf("rebuilt=%v err=%v", rebuilt, err)
	}
	s.Close()
	if broken, _ := filepath.Glob(path + ".broken-*"); len(broken) != 1 {
		t.Fatalf("broken = %v", broken)
	}
}

func TestRepositories(t *testing.T) {
	s, _, err := Open(filepath.Join(t.TempDir(), "stickypane.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	at := time.Date(2026, 10, 28, 14, 2, 11, 0, time.FixedZone("KST", 9*3600))
	var id int64
	err = s.Write(func(r ports.Repos) error {
		var err error
		if id, err = r.Projects.Create(model.Project{Path: "/w", Title: "w", FirstSeen: at, LastSeen: at}); err != nil {
			return err
		}
		if err := r.Sessions.Start(id, 7, "tui", at); err != nil {
			return err
		}
		if err := r.Sessions.Start(id, 7, "bridge", at); err != nil { // already there: kept
			return err
		}
		if err := r.Ingest.SetOffset("d.jsonl", 10); err != nil {
			return err
		}
		if err := r.Ingest.SetOffset("d.jsonl", 20); err != nil {
			return err
		}
		if err := r.Ingest.AddSkipped(2); err != nil {
			return err
		}
		return r.Ingest.AddSkipped(3)
	})
	if err != nil {
		t.Fatal(err)
	}
	r := s.Repos()
	if p, ok, _ := r.Projects.ByPath("/w"); !ok || p.ID != id || !p.FirstSeen.Equal(at) {
		t.Fatalf("project = %+v", p)
	}
	if off, _ := r.Ingest.Offset("d.jsonl"); off != 20 {
		t.Fatalf("offset = %d", off)
	}
	if n, _ := r.Ingest.Skipped(); n != 5 {
		t.Fatalf("skipped = %d", n)
	}
	r.Sessions.EndPID(7, at.Add(time.Hour))
	var kind string
	var ended *string
	s.DB().Table("session").Select("kind, ended_at").Where("pid = ?", 7).Row().Scan(&kind, &ended)
	if kind != "tui" || ended == nil {
		t.Fatalf("kind=%q ended=%v", kind, ended)
	}
	removed := at
	r.Projects.SetRemoved(id, &removed)
	if p, _, _ := r.Projects.ByPath("/w"); p.RemovedAt == nil {
		t.Fatal("not removed")
	}
	r.Projects.SetRemoved(id, nil)
	if p, _, _ := r.Projects.ByPath("/w"); p.RemovedAt != nil {
		t.Fatal("still removed")
	}
	// A failed unit of work leaves nothing behind.
	s.Write(func(r ports.Repos) error {
		r.Ingest.SetOffset("e.jsonl", 5)
		return os.ErrInvalid
	})
	if off, _ := r.Ingest.Offset("e.jsonl"); off != 0 {
		t.Fatal("rolled-back write kept")
	}
}
