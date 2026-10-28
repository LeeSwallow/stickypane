// Package model is the domain of the board index: what a project, a
// session and a day of work are, the records stickypane leaves in
// ~/.stickypane as values, and the rules that turn them into the home
// list (docs/overlay/08 §8.4). It is pure: no files, no database, no
// processes, no clock. Those are reached through ../ports.
package model

import (
	"path/filepath"
	"time"
)

// Rules of the home list and of what is kept.
const (
	// ActiveWindow is how recently .sticky/ must have changed for a project
	// to count as active without a live process.
	ActiveWindow = 30 * time.Minute
	// KeepRemoved is how long the row of a project whose folder is gone is
	// kept, so that it can be restored.
	KeepRemoved = 30 * 24 * time.Hour
)

// Project is one board folder seen on this machine.
type Project struct {
	ID        int64
	Path      string
	Title     string
	FirstSeen time.Time
	LastSeen  time.Time
	RemovedAt *time.Time // nil: not removed
}

// Session is a live stickypane process on a project.
type Session struct {
	PID     int
	Kind    string
	Started time.Time
}

// Row is one line of the home list. Agents (the @assignees of the board's
// cards) are not known to the index; the caller fills them from the bridge.
type Row struct {
	Project
	Agents []string
	Live   []Session
	LastAt time.Time
}

// Day is the rows whose last activity fell on one local date.
type Day struct {
	Date string // YYYY-MM-DD
	Rows []Row
}

// Groups is the home list: what is moving now, the rest by day, and what
// was just tidied away.
type Groups struct {
	Active  []Row
	Days    []Day
	Removed []Project
}

// DayStat is what the index keeps of one project on one local date.
type DayStat struct {
	Date    string
	Events  int
	FirstAt time.Time
	LastAt  time.Time
}

// CleanPath is the form a project's path is kept in.
func CleanPath(p string) string { return filepath.Clean(p) }

// DefaultTitle is a project's title until its board says another: the
// folder's name.
func DefaultTitle(path string) string { return filepath.Base(path) }

// LocalDate is the local date of t, as a day of the index is named.
//
// Known limit: the day is the local date of the event time in the zone in
// force now; an event written in the repeated hour of a DST change can
// fall on the neighbouring date.
func LocalDate(t time.Time) string { return t.In(time.Local).Format("2006-01-02") }

// Count adds one event at at to d (a zero DayStat is a day not seen yet).
func (d DayStat) Count(at time.Time) DayStat {
	if d.Events == 0 || d.FirstAt.IsZero() || at.Before(d.FirstAt) {
		d.FirstAt = at
	}
	if d.Events == 0 || d.LastAt.IsZero() || at.After(d.LastAt) {
		d.LastAt = at
	}
	d.Events++
	return d
}

// Later says whether at moves a project's last activity from last.
func Later(last, at time.Time) bool { return last.IsZero() || at.After(last) }
