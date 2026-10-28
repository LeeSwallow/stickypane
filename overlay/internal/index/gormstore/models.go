// Package gormstore is the index's database: stickypane.db, SQLite in WAL
// mode through GORM and the pure-Go driver github.com/glebarez/sqlite. It
// implements the repositories and the unit of work of ../ports; the rows
// here are the tables of docs/overlay/08 §8.4.3, and times are kept as
// RFC 3339 text, as the first version of the file did.
package gormstore

// projectRow is a row of project.
type projectRow struct {
	ID        int64   `gorm:"column:id"`
	Path      string  `gorm:"column:path"`
	Title     *string `gorm:"column:title"`
	FirstSeen string  `gorm:"column:first_seen"`
	LastSeen  string  `gorm:"column:last_seen"`
	RemovedAt *string `gorm:"column:removed_at"`
}

func (projectRow) TableName() string { return "project" }

// dayRow is a row of day.
type dayRow struct {
	ProjectID int64  `gorm:"column:project_id;primaryKey;autoIncrement:false"`
	Date      string `gorm:"column:date;primaryKey"`
	Events    int    `gorm:"column:events"`
	FirstAt   string `gorm:"column:first_at"`
	LastAt    string `gorm:"column:last_at"`
}

func (dayRow) TableName() string { return "day" }

// sessionRow is a row of session.
type sessionRow struct {
	PID       int     `gorm:"column:pid;primaryKey;autoIncrement:false"`
	ProjectID int64   `gorm:"column:project_id"`
	Kind      string  `gorm:"column:kind"`
	StartedAt string  `gorm:"column:started_at;primaryKey"`
	EndedAt   *string `gorm:"column:ended_at"`
}

func (sessionRow) TableName() string { return "session" }

// ingestRow is a row of ingest: how far an events file has been read.
type ingestRow struct {
	File   string `gorm:"column:file;primaryKey"`
	Offset int64  `gorm:"column:offset"`
}

func (ingestRow) TableName() string { return "ingest" }

// metaRow is a row of meta (schema_version, skipped_lines).
type metaRow struct {
	Key   string `gorm:"column:key;primaryKey"`
	Value string `gorm:"column:value"`
}

func (metaRow) TableName() string { return "meta" }
