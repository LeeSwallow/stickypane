package model

import (
	"reflect"
	"testing"
	"time"
)

func TestParseEventNeedsATypeAProjectAndATime(t *testing.T) {
	e, at, ok := ParseEvent([]byte(`{"v":1,"at":"2026-10-28T14:02:11+09:00","type":"session.start","project":"/w","pid":7,"kind":"bridge"}`))
	if !ok || e.PID != 7 || e.SessionKind() != "bridge" || at.Hour() != 14 {
		t.Fatalf("e=%+v at=%v ok=%v", e, at, ok)
	}
	for _, bad := range []string{
		`{not json`,
		`{"v":1,"type":"project.touch","at":"2026-10-28T14:02:11+09:00"}`,
		`{"v":1,"project":"/w","at":"2026-10-28T14:02:11+09:00"}`,
		`{"v":1,"type":"project.touch","project":"/w","at":"yesterday"}`,
	} {
		if _, _, ok := ParseEvent([]byte(bad)); ok {
			t.Errorf("%s parsed", bad)
		}
	}
	if (Event{}).SessionKind() != "tui" {
		t.Error("a session without a kind is not tui")
	}
}

func TestLinesLeavesAPartialLastLine(t *testing.T) {
	lines, used := Lines([]byte("a\n\n  \nb\nc"))
	if used != 8 || !reflect.DeepEqual(lines, [][]byte{[]byte("a"), []byte("b")}) {
		t.Fatalf("lines=%q used=%d", lines, used)
	}
	if lines, used := Lines([]byte("partial")); lines != nil || used != 0 {
		t.Fatalf("lines=%q used=%d", lines, used)
	}
}

func TestRunFiles(t *testing.T) {
	for name, want := range map[string]bool{"4242.json": true, "touch-abc": false, ".tmp-1": false, "x.json": false} {
		if IsRunName(name) != want {
			t.Errorf("IsRunName(%q) != %v", name, want)
		}
	}
	if _, _, ok := ParseRun([]byte(`{"pid":0,"started":"2026-10-28T14:02:11+09:00"}`)); ok {
		t.Error("pid 0 parsed")
	}
	if _, _, ok := ParseRun([]byte(`{"pid":3}`)); ok {
		t.Error("no start time parsed")
	}
}

func TestDayCountKeepsTheFirstAndLast(t *testing.T) {
	base := time.Date(2026, 10, 28, 12, 0, 0, 0, time.UTC)
	var d DayStat
	for _, m := range []int{10, -5, 30, 0} {
		d = d.Count(base.Add(time.Duration(m) * time.Minute))
	}
	if d.Events != 4 || !d.FirstAt.Equal(base.Add(-5*time.Minute)) || !d.LastAt.Equal(base.Add(30*time.Minute)) {
		t.Fatalf("%+v", d)
	}
}

func TestSettleOf(t *testing.T) {
	now := time.Now()
	gone := now.Add(-31 * 24 * time.Hour)
	lately := now.Add(-time.Hour)
	for _, c := range []struct {
		removed *time.Time
		exists  bool
		want    Settle
	}{
		{nil, true, Keep},
		{nil, false, Remove},
		{&lately, true, Restore},
		{&lately, false, Keep},
		{&gone, false, Forget},
		{&gone, true, Restore},
	} {
		if got := SettleOf(Project{RemovedAt: c.removed}, c.exists, now); got != c.want {
			t.Errorf("removed=%v exists=%v: %v, want %v", c.removed != nil, c.exists, got, c.want)
		}
	}
}

func TestGroup(t *testing.T) {
	day := time.Date(2026, 10, 28, 12, 0, 0, 0, time.Local)
	removed := day
	ps := []Project{
		{ID: 1, Path: "/live", LastSeen: day.AddDate(0, 0, -5)},
		{ID: 2, Path: "/recent", LastSeen: day.AddDate(0, 0, -5)},
		{ID: 3, Path: "/old", LastSeen: day.AddDate(0, 0, -1)},
		{ID: 4, Path: "/older", LastSeen: day.AddDate(0, 0, -2)},
		{ID: 5, Path: "/older2", LastSeen: day.AddDate(0, 0, -2).Add(time.Hour)},
		{ID: 6, Path: "/gone", RemovedAt: &removed},
	}
	g := Group(ps, map[string][]Session{"/live": {{PID: 9}}}, map[string]bool{"/recent": true})
	if len(g.Active) != 2 || len(g.Removed) != 1 || g.Removed[0].ID != 6 {
		t.Fatalf("%+v", g)
	}
	if len(g.Days) != 2 || g.Days[0].Rows[0].ID != 3 || g.Days[1].Rows[0].ID != 5 || g.Days[1].Rows[1].ID != 4 {
		t.Fatalf("days = %+v", g.Days)
	}
	if !Recent(day.Add(-ActiveWindow), day) || Recent(day.Add(-ActiveWindow-time.Second), day) {
		t.Error("ActiveWindow edge")
	}
}
