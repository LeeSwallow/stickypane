package model

import (
	"sort"
	"time"
)

// Settle is what to do with a project row when the index checks it
// against the disk (08 §8.4.4).
type Settle int

const (
	Keep    Settle = iota // nothing changes
	Remove                // the folder is gone: mark it removed now
	Restore               // the folder is back: clear the mark
	Forget                // removed long enough ago: delete the row
)

// SettleOf decides for p, whose board exists or not, at now.
func SettleOf(p Project, exists bool, now time.Time) Settle {
	switch {
	case !exists && p.RemovedAt == nil:
		return Remove
	case exists && p.RemovedAt != nil:
		return Restore
	case !exists && now.Sub(*p.RemovedAt) > KeepRemoved:
		return Forget
	}
	return Keep
}

// Group builds the home list. live is the sessions of live processes by
// project path; recent says which project paths had .sticky/ change
// within ActiveWindow. Active is a project with a live session or a recent
// change; the rest are grouped by the local date of their last activity,
// newest first, one row per project. Removed projects are listed apart.
func Group(ps []Project, live map[string][]Session, recent map[string]bool) Groups {
	var g Groups
	days := map[string][]Row{}
	for _, p := range ps {
		if p.RemovedAt != nil {
			g.Removed = append(g.Removed, p)
			continue
		}
		row := Row{Project: p, Live: live[p.Path], LastAt: p.LastSeen}
		if len(row.Live) > 0 || recent[p.Path] {
			g.Active = append(g.Active, row)
			continue
		}
		d := LocalDate(row.LastAt)
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
	return g
}

// Recent says whether a board last changed at changed is active at now.
func Recent(changed, now time.Time) bool { return now.Sub(changed) <= ActiveWindow }
