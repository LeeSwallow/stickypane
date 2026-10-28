package service

import (
	"strings"
	"testing"
	"time"

	"github.com/LeeSwallow/stickypane/desktop/internal/index/model"
	"github.com/LeeSwallow/stickypane/desktop/internal/index/ports"
)

// The use cases on ports kept in memory: no database, no files.

type mem struct {
	projects []model.Project
	days     map[[2]any]model.DayStat
	sessions map[[2]any]*struct {
		project int64
		ended   bool
	}
	offsets map[string]int64
	skipped int
}

func newMem() *mem {
	return &mem{days: map[[2]any]model.DayStat{}, sessions: map[[2]any]*struct {
		project int64
		ended   bool
	}{}, offsets: map[string]int64{}}
}

func (m *mem) Repos() ports.Repos {
	return ports.Repos{Projects: memProjects{m}, Days: memDays{m}, Sessions: memSessions{m}, Ingest: memIngest{m}}
}
func (m *mem) Write(f func(ports.Repos) error) error { return f(m.Repos()) }

type memProjects struct{ m *mem }

func (r memProjects) All() ([]model.Project, error) {
	return append([]model.Project(nil), r.m.projects...), nil
}
func (r memProjects) ByPath(path string) (model.Project, bool, error) {
	for _, p := range r.m.projects {
		if p.Path == path {
			return p, true, nil
		}
	}
	return model.Project{}, false, nil
}
func (r memProjects) Create(p model.Project) (int64, error) {
	p.ID = int64(len(r.m.projects) + 1)
	r.m.projects = append(r.m.projects, p)
	return p.ID, nil
}
func (r memProjects) each(f func(*model.Project)) {
	for i := range r.m.projects {
		f(&r.m.projects[i])
	}
}
func (r memProjects) SetLastSeen(id int64, t time.Time) error {
	r.each(func(p *model.Project) {
		if p.ID == id {
			p.LastSeen = t
		}
	})
	return nil
}
func (r memProjects) SetRemoved(id int64, at *time.Time) error {
	r.each(func(p *model.Project) {
		if p.ID == id {
			p.RemovedAt = at
		}
	})
	return nil
}
func (r memProjects) SetTitle(path, title string) error {
	r.each(func(p *model.Project) {
		if p.Path == path {
			p.Title = title
		}
	})
	return nil
}
func (r memProjects) Delete(id int64) error {
	var keep []model.Project
	for _, p := range r.m.projects {
		if p.ID != id {
			keep = append(keep, p)
		}
	}
	r.m.projects = keep
	return nil
}

type memDays struct{ m *mem }

func (r memDays) Get(id int64, date string) (model.DayStat, bool, error) {
	d, ok := r.m.days[[2]any{id, date}]
	return d, ok, nil
}
func (r memDays) Put(id int64, d model.DayStat) error { r.m.days[[2]any{id, d.Date}] = d; return nil }

type memSessions struct{ m *mem }

func (r memSessions) Start(project int64, pid int, kind string, at time.Time) error {
	k := [2]any{pid, at.Unix()}
	if r.m.sessions[k] == nil {
		r.m.sessions[k] = &struct {
			project int64
			ended   bool
		}{project: project}
	}
	return nil
}
func (r memSessions) End(project int64, pid int, at time.Time) error {
	for k, s := range r.m.sessions {
		if k[0] == pid && s.project == project {
			s.ended = true
		}
	}
	return nil
}
func (r memSessions) EndPID(pid int, at time.Time) error {
	for k, s := range r.m.sessions {
		if k[0] == pid {
			s.ended = true
		}
	}
	return nil
}

type memIngest struct{ m *mem }

func (r memIngest) Offset(f string) (int64, error)      { return r.m.offsets[f], nil }
func (r memIngest) SetOffset(f string, off int64) error { r.m.offsets[f] = off; return nil }
func (r memIngest) Files() ([]string, error) {
	var fs []string
	for f := range r.m.offsets {
		fs = append(fs, f)
	}
	return fs, nil
}
func (r memIngest) Forget(f string) error  { delete(r.m.offsets, f); return nil }
func (r memIngest) AddSkipped(n int) error { r.m.skipped += n; return nil }
func (r memIngest) Skipped() (int, error)  { return r.m.skipped, nil }

// records is a home folder in memory.
type records struct {
	events map[string]string
	runs   []ports.RunFile
}

func (r records) EventFiles() ([]string, error) {
	var fs []string
	for f := range r.events {
		fs = append(fs, f)
	}
	return fs, nil
}
func (r records) ReadFrom(name string, off int64) ([]byte, int64, error) {
	s, ok := r.events[name]
	if !ok {
		return nil, 0, ports.ErrGone
	}
	if int64(len(s)) < off {
		return nil, int64(len(s)), nil
	}
	return []byte(s[off:]), int64(len(s)), nil
}
func (r records) Runs() ([]ports.RunFile, error) { return r.runs, nil }

type boards map[string]bool

func (b boards) Exists(p string) bool               { return b[p] }
func (b boards) Changed(p string) (time.Time, bool) { return time.Time{}, b[p] }

type clock time.Time

func (c clock) Now() time.Time { return time.Time(c) }

type alive map[int]bool

func (a alive) Alive(pid int, _ time.Time) bool { return a[pid] }

func TestServiceOnMemoryPorts(t *testing.T) {
	now := time.Date(2026, 10, 28, 14, 0, 0, 0, time.Local)
	at := now.Format(time.RFC3339)
	lines := []string{
		`{"v":1,"at":"` + at + `","type":"session.start","project":"/w/a/","pid":7,"kind":"tui"}`,
		`{broken`,
		`{"v":1,"at":"` + at + `","type":"project.touch","project":"/w/b","cmd":"card"}`,
		`{"v":1,"at":"` + at + `","type":"project.touch","project":"/w/a"`, // no newline yet
	}
	rec := records{events: map[string]string{"d.jsonl": strings.Join(lines, "\n")}, runs: []ports.RunFile{
		{Name: "7.json", Data: []byte(`{"kind":"tui","pid":7,"project":"/w/a","started":"` + at + `"}`)},
		{Name: "touch-1", Data: nil},
	}}
	m := newMem()
	s := &Service{Store: m, Records: rec, Boards: boards{"/w/a": true}, Clock: clock(now), Procs: alive{7: true}}
	if err := s.Ingest(); err != nil {
		t.Fatal(err)
	}
	if len(m.projects) != 2 || m.projects[0].Path != "/w/a" || m.projects[0].Title != "a" || s.Skipped() != 1 {
		t.Fatalf("projects=%+v skipped=%d", m.projects, s.Skipped())
	}
	g, _ := s.Sessions(now)
	if len(g.Active) != 1 || g.Active[0].Live[0].PID != 7 || len(g.Days) != 1 {
		t.Fatalf("groups = %+v", g)
	}
	removed, _ := s.Prune(now)
	if len(removed) != 1 || removed[0].Path != "/w/b" {
		t.Fatalf("removed = %+v", removed)
	}
	if _, err := s.Prune(now.Add(model.KeepRemoved + time.Hour)); err != nil || len(m.projects) != 1 {
		t.Fatalf("not forgotten: %+v", m.projects)
	}
	s.Procs = alive{}
	s.Ingest()
	for _, sess := range m.sessions {
		if !sess.ended {
			t.Fatal("dead pid's session open")
		}
	}
	s.Retitle("/w/a/", "Alpha")
	if m.projects[0].Title != "Alpha" {
		t.Fatalf("title = %q", m.projects[0].Title)
	}
	delete(rec.events, "d.jsonl")
	s.Records = rec
	s.Ingest()
	if len(m.offsets) != 0 {
		t.Fatalf("offsets of a gone file kept: %v", m.offsets)
	}
}
