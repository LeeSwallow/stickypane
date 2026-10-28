package gormstore

import (
	"strconv"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// SchemaVersion is the version written to meta.schema_version.
const SchemaVersion = 1

// migrations[v] takes the database from version v to v+1, in the write
// transaction of Open. Each is idempotent: it only makes what is missing.
var migrations = []func(tx *gorm.DB) error{
	createV1,
}

// The tables of version 1, as they are made. The rows of models.go are
// what the repositories read and write; these carry the declarations
// (types, keys, the foreign keys that cascade a deleted project).
type (
	projectV1 struct {
		ID        int64   `gorm:"column:id;type:INTEGER;primaryKey"`
		Path      string  `gorm:"column:path;type:TEXT;not null;unique"`
		Title     *string `gorm:"column:title;type:TEXT"`
		FirstSeen string  `gorm:"column:first_seen;type:TEXT;not null"`
		LastSeen  string  `gorm:"column:last_seen;type:TEXT;not null"`
		RemovedAt *string `gorm:"column:removed_at;type:TEXT"`
	}
	dayV1 struct {
		ProjectID int64     `gorm:"column:project_id;type:INTEGER;not null;primaryKey;autoIncrement:false"`
		Project   projectV1 `gorm:"foreignKey:ProjectID;references:ID;constraint:OnDelete:CASCADE"`
		Date      string    `gorm:"column:date;type:TEXT;not null;primaryKey"`
		Events    int       `gorm:"column:events;type:INTEGER;not null;default:0"`
		FirstAt   string    `gorm:"column:first_at;type:TEXT;not null"`
		LastAt    string    `gorm:"column:last_at;type:TEXT;not null"`
	}
	sessionV1 struct {
		PID       int       `gorm:"column:pid;type:INTEGER;not null;primaryKey;autoIncrement:false"`
		ProjectID int64     `gorm:"column:project_id;type:INTEGER;not null"`
		Project   projectV1 `gorm:"foreignKey:ProjectID;references:ID;constraint:OnDelete:CASCADE"`
		Kind      string    `gorm:"column:kind;type:TEXT;not null"`
		StartedAt string    `gorm:"column:started_at;type:TEXT;not null;primaryKey"`
		EndedAt   *string   `gorm:"column:ended_at;type:TEXT"`
	}
	ingestV1 struct {
		File   string `gorm:"column:file;type:TEXT;primaryKey"`
		Offset int64  `gorm:"column:offset;type:INTEGER;not null"`
	}
	metaV1 struct {
		Key   string `gorm:"column:key;type:TEXT;primaryKey"`
		Value string `gorm:"column:value;type:TEXT"`
	}
)

func (projectV1) TableName() string { return "project" }
func (dayV1) TableName() string     { return "day" }
func (sessionV1) TableName() string { return "session" }
func (ingestV1) TableName() string  { return "ingest" }
func (metaV1) TableName() string    { return "meta" }

func createV1(tx *gorm.DB) error {
	m := tx.Migrator()
	for _, t := range []any{&projectV1{}, &dayV1{}, &sessionV1{}, &ingestV1{}, &metaV1{}} {
		if m.HasTable(t) {
			continue
		}
		if err := m.CreateTable(t); err != nil {
			return err
		}
	}
	return nil
}

// migrate brings the database to SchemaVersion inside tx. A database of a
// later version, or one whose version cannot be read, is damaged for this
// app: it is set aside and made again from events/.
func migrate(tx *gorm.DB) error {
	v := 0
	if tx.Migrator().HasTable(&metaV1{}) {
		var rows []metaRow
		if err := tx.Where("key = ?", "schema_version").Limit(1).Find(&rows).Error; err != nil {
			return err
		}
		if len(rows) == 1 {
			n, err := strconv.Atoi(rows[0].Value)
			if err != nil || n < 0 || n > SchemaVersion {
				return &corruptError{"schema version " + rows[0].Value}
			}
			v = n
		}
	}
	for ; v < SchemaVersion; v++ {
		if err := migrations[v](tx); err != nil {
			return err
		}
	}
	return tx.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "key"}},
		DoUpdates: clause.AssignmentColumns([]string{"value"}),
	}).Create(&metaRow{Key: "schema_version", Value: strconv.Itoa(SchemaVersion)}).Error
}
