package gormstore

import (
	"strconv"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/LeeSwallow/stickypane/overlay/internal/index/model"
	"github.com/LeeSwallow/stickypane/overlay/internal/index/ports"
)

func reposOf(db *gorm.DB) ports.Repos {
	return ports.Repos{
		Projects: projects{db},
		Days:     days{db},
		Sessions: sessions{db},
		Ingest:   ingest{db},
	}
}

// stamp is how a time is kept: RFC 3339, in the zone it was given in.
func stamp(t time.Time) string { return t.Format(time.RFC3339) }

func parse(s string) time.Time {
	t, _ := time.Parse(time.RFC3339, s)
	return t
}

func ptr(s string) *string { return &s }

// projects is the ProjectRepository.
type projects struct{ db *gorm.DB }

func (r projects) toModel(row projectRow) model.Project {
	p := model.Project{ID: row.ID, Path: row.Path, FirstSeen: parse(row.FirstSeen), LastSeen: parse(row.LastSeen)}
	if row.Title != nil {
		p.Title = *row.Title
	}
	if row.RemovedAt != nil {
		if t, err := time.Parse(time.RFC3339, *row.RemovedAt); err == nil {
			p.RemovedAt = &t
		}
	}
	return p
}

func (r projects) All() ([]model.Project, error) {
	var rows []projectRow
	if err := r.db.Order("id").Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]model.Project, 0, len(rows))
	for _, row := range rows {
		out = append(out, r.toModel(row))
	}
	return out, nil
}

func (r projects) ByPath(path string) (model.Project, bool, error) {
	var rows []projectRow
	if err := r.db.Where("path = ?", path).Limit(1).Find(&rows).Error; err != nil || len(rows) == 0 {
		return model.Project{}, false, err
	}
	return r.toModel(rows[0]), true, nil
}

func (r projects) Create(p model.Project) (int64, error) {
	row := projectRow{Path: p.Path, Title: ptr(p.Title), FirstSeen: stamp(p.FirstSeen), LastSeen: stamp(p.LastSeen)}
	if p.RemovedAt != nil {
		row.RemovedAt = ptr(stamp(*p.RemovedAt))
	}
	err := r.db.Create(&row).Error
	return row.ID, err
}

func (r projects) SetLastSeen(id int64, t time.Time) error {
	return r.db.Model(&projectRow{}).Where("id = ?", id).Update("last_seen", stamp(t)).Error
}

func (r projects) SetRemoved(id int64, at *time.Time) error {
	var v *string
	if at != nil {
		v = ptr(stamp(*at))
	}
	return r.db.Model(&projectRow{}).Where("id = ?", id).Update("removed_at", v).Error
}

func (r projects) SetTitle(path, title string) error {
	return r.db.Model(&projectRow{}).Where("path = ?", path).Update("title", title).Error
}

func (r projects) Delete(id int64) error {
	// The foreign keys (foreign_keys on) take its days and sessions.
	return r.db.Where("id = ?", id).Delete(&projectRow{}).Error
}

// days is the DayRepository.
type days struct{ db *gorm.DB }

func (r days) Get(projectID int64, date string) (model.DayStat, bool, error) {
	var rows []dayRow
	err := r.db.Where("project_id = ? AND date = ?", projectID, date).Limit(1).Find(&rows).Error
	if err != nil || len(rows) == 0 {
		return model.DayStat{}, false, err
	}
	d := rows[0]
	return model.DayStat{Date: d.Date, Events: d.Events, FirstAt: parse(d.FirstAt), LastAt: parse(d.LastAt)}, true, nil
}

func (r days) Put(projectID int64, d model.DayStat) error {
	return r.db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "project_id"}, {Name: "date"}},
		DoUpdates: clause.AssignmentColumns([]string{"events", "first_at", "last_at"}),
	}).Create(&dayRow{ProjectID: projectID, Date: d.Date, Events: d.Events, FirstAt: stamp(d.FirstAt), LastAt: stamp(d.LastAt)}).Error
}

// sessions is the SessionRepository.
type sessions struct{ db *gorm.DB }

func (r sessions) Start(projectID int64, pid int, kind string, at time.Time) error {
	return r.db.Clauses(clause.OnConflict{DoNothing: true}).
		Create(&sessionRow{PID: pid, ProjectID: projectID, Kind: kind, StartedAt: stamp(at)}).Error
}

func (r sessions) End(projectID int64, pid int, at time.Time) error {
	return r.db.Model(&sessionRow{}).
		Where("pid = ? AND project_id = ? AND ended_at IS NULL", pid, projectID).
		Update("ended_at", stamp(at)).Error
}

func (r sessions) EndPID(pid int, at time.Time) error {
	return r.db.Model(&sessionRow{}).
		Where("pid = ? AND ended_at IS NULL", pid).
		Update("ended_at", stamp(at)).Error
}

// ingest is the IngestLog.
type ingest struct{ db *gorm.DB }

func (r ingest) Offset(file string) (int64, error) {
	var rows []ingestRow
	if err := r.db.Where("file = ?", file).Limit(1).Find(&rows).Error; err != nil || len(rows) == 0 {
		return 0, err
	}
	return rows[0].Offset, nil
}

func (r ingest) SetOffset(file string, offset int64) error {
	return r.db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "file"}},
		DoUpdates: clause.AssignmentColumns([]string{"offset"}),
	}).Create(&ingestRow{File: file, Offset: offset}).Error
}

func (r ingest) Files() ([]string, error) {
	var files []string
	err := r.db.Model(&ingestRow{}).Order("file").Pluck("file", &files).Error
	return files, err
}

func (r ingest) Forget(file string) error {
	return r.db.Where("file = ?", file).Delete(&ingestRow{}).Error
}

func (r ingest) AddSkipped(n int) error {
	return r.db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "key"}},
		DoUpdates: clause.Assignments(map[string]any{"value": gorm.Expr("CAST(value AS INTEGER) + ?", n)}),
	}).Create(&metaRow{Key: "skipped_lines", Value: strconv.Itoa(n)}).Error
}

func (r ingest) Skipped() (int, error) {
	var rows []metaRow
	if err := r.db.Where("key = ?", "skipped_lines").Limit(1).Find(&rows).Error; err != nil || len(rows) == 0 {
		return 0, err
	}
	n, _ := strconv.Atoi(rows[0].Value)
	return n, nil
}
