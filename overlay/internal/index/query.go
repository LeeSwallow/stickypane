package index

import (
	"database/sql"
	"os"
	"path/filepath"
	"sort"
	"time"
)

// activeWindow is how recently .sticky/ must have changed for a project to
// count as active without a live process.
const activeWindow = 30 * time.Minute

// Project is a row of the project table.
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

// Projects returns every project row, oldest first.
func (ix *Index) Projects() ([]Project, error) {
	rows, err := ix.db.Query(`SELECT id, path, COALESCE(title,''), first_seen, last_seen, removed_at FROM project ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Project
	for rows.Next() {
		var p Project
		var first, last string
		var removed sql.NullString
		if err := rows.Scan(&p.ID, &p.Path, &p.Title, &first, &last, &removed); err != nil {
			return nil, err
		}
		p.FirstSeen, _ = time.Parse(time.RFC3339, first)
		p.LastSeen, _ = time.Parse(time.RFC3339, last)
		if removed.Valid {
			if t, err := time.Parse(time.RFC3339, removed.String); err == nil {
				p.RemovedAt = &t
			}
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// Sessions builds the home list at now. Active is a live run file (process
// alive and started when the file says) or a .sticky/ changed within 30
// minutes; the rest are grouped by the local date of their last activity,
// newest first, one row per project. Removed projects are listed apart.
func (ix *Index) Sessions(now time.Time) (Groups, error) {
	var g Groups
	ps, err := ix.Projects()
	if err != nil {
		return g, err
	}
	runs, err := ix.liveRuns(false)
	if err != nil {
		return g, err
	}
	liveBy := map[string][]Session{}
	for _, r := range runs {
		started, _ := time.Parse(time.RFC3339, r.Started)
		p := filepath.Clean(r.Project)
		liveBy[p] = append(liveBy[p], Session{PID: r.PID, Kind: r.Kind, Started: started})
	}
	days := map[string][]Row{}
	for _, p := range ps {
		if p.RemovedAt != nil {
			g.Removed = append(g.Removed, p)
			continue
		}
		row := Row{Project: p, Live: liveBy[p.Path], LastAt: p.LastSeen}
		recent := false
		if fi, err := os.Stat(filepath.Join(p.Path, ".sticky")); err == nil {
			recent = now.Sub(fi.ModTime()) <= activeWindow
		}
		if len(row.Live) > 0 || recent {
			g.Active = append(g.Active, row)
			continue
		}
		d := row.LastAt.In(time.Local).Format("2006-01-02")
		days[d] = append(days[d], row)
	}
	byLast := func(rs []Row) {
		sort.SliceStable(rs, func(i, j int) bool { return rs[i].LastAt.After(rs[j].LastAt) })
	}
	byLast(g.Active)
	for d, rs := range days {
		byLast(rs)
		g.Days = append(g.Days, Day{Date: d, Rows: rs})
	}
	sort.Slice(g.Days, func(i, j int) bool { return g.Days[i].Date > g.Days[j].Date })
	return g, nil
}
