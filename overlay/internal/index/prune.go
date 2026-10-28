package index

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	keepRemoved = 30 * 24 * time.Hour // rows of removed projects
	keepEvents  = 90 * 24 * time.Hour // events/ files
)

// boardExists says whether the project folder and its .sticky/ are there.
func boardExists(path string) bool {
	fi, err := os.Stat(filepath.Join(path, ".sticky"))
	return err == nil && fi.IsDir()
}

// projectTitle is the title in .sticky/sticky.json, else the folder name.
func projectTitle(path string) string {
	if b, err := os.ReadFile(filepath.Join(path, ".sticky", "sticky.json")); err == nil {
		var s struct {
			Title string `json:"title"`
		}
		if json.Unmarshal(b, &s) == nil && strings.TrimSpace(s.Title) != "" {
			return strings.TrimSpace(s.Title)
		}
	}
	return filepath.Base(path)
}

// Prune settles what has been deleted (08 §8.4.4). A project whose folder or
// .sticky/ is gone is marked removed at now and returned (only once); one
// whose folder is back is restored; titles are refreshed; rows removed more
// than 30 days ago are deleted; event files older than 90 days are removed.
// The day table is kept.
func (ix *Index) Prune(now time.Time) (removed []Project, err error) {
	ps, err := ix.Projects()
	if err != nil {
		return nil, err
	}
	for _, p := range ps {
		switch {
		case !boardExists(p.Path) && p.RemovedAt == nil:
			if _, err := ix.db.Exec(`UPDATE project SET removed_at=? WHERE id=?`, stamp(now), p.ID); err != nil {
				return nil, err
			}
			t := now
			p.RemovedAt = &t
			removed = append(removed, p)
		case boardExists(p.Path):
			if _, err := ix.db.Exec(`UPDATE project SET removed_at=NULL, title=? WHERE id=?`, projectTitle(p.Path), p.ID); err != nil {
				return nil, err
			}
		case p.RemovedAt != nil && now.Sub(*p.RemovedAt) > keepRemoved:
			if _, err := ix.db.Exec(`DELETE FROM project WHERE id=?`, p.ID); err != nil {
				return nil, err
			}
		}
	}
	files, _ := filepath.Glob(filepath.Join(ix.home, "events", "*.jsonl"))
	for _, f := range files {
		name := filepath.Base(f)
		d, perr := time.ParseInLocation("2006-01-02", strings.TrimSuffix(name, ".jsonl"), time.Local)
		if perr != nil || now.Sub(d) <= keepEvents {
			continue
		}
		if os.Remove(f) == nil {
			if _, err := ix.db.Exec(`DELETE FROM ingest WHERE file=?`, name); err != nil {
				return removed, err
			}
		}
	}
	return removed, nil
}

// Restore undoes the removal of project id (the "undo" of the home screen).
func (ix *Index) Restore(id int64) error {
	_, err := ix.db.Exec(`UPDATE project SET removed_at=NULL WHERE id=?`, id)
	return err
}
