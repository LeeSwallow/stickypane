package gormstore

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/LeeSwallow/stickypane/desktop/internal/index/ports"
)

// Store is the open database.
type Store struct{ db *gorm.DB }

var _ ports.Store = (*Store)(nil)

// corruptError marks a database that must be moved aside and made again:
// one that fails quick_check, is not a database file, or has a schema
// version this app does not know. Every other failure (BUSY, permissions,
// I/O) is returned as it is and never costs the user the file.
type corruptError struct{ why string }

func (e *corruptError) Error() string { return "index: damaged database: " + e.why }

// looksCorrupt says whether a SQLite error means a damaged file.
func looksCorrupt(err error) bool {
	m := strings.ToLower(err.Error())
	return strings.Contains(m, "file is not a database") ||
		strings.Contains(m, "malformed") || strings.Contains(m, "(26)") || strings.Contains(m, "(11)")
}

func isBusy(err error) bool {
	m := err.Error()
	return strings.Contains(m, "SQLITE_BUSY") || strings.Contains(m, "locked")
}

func wrap(err error) error {
	var ce *corruptError
	if err == nil || errors.As(err, &ce) {
		return err
	}
	if looksCorrupt(err) {
		return &corruptError{err.Error()}
	}
	return err
}

// Open opens (making it if need be) the database at path. One that is
// damaged, or of a version this app does not know, is renamed to
// path.broken-<time> and made again empty; rebuilt says so, and the
// caller then reads every record again.
func Open(path string) (s *Store, rebuilt bool, err error) {
	s, err = openFile(path)
	if err == nil {
		return s, false, nil
	}
	var ce *corruptError
	if !errors.As(err, &ce) {
		return nil, false, err
	}
	stamp := time.Now().Format("20060102-150405.000")
	if rerr := os.Rename(path, path+".broken-"+stamp); rerr != nil {
		return nil, false, fmt.Errorf("%w (and could not move it aside: %v)", err, rerr)
	}
	os.Remove(path + "-wal")
	os.Remove(path + "-shm")
	s, err = openFile(path)
	if err != nil {
		return nil, false, err
	}
	return s, true, nil
}

func openFile(path string) (*Store, error) {
	// _txlock=immediate: every transaction takes the write lock at BEGIN, so
	// two processes never read stale state and then fail with BUSY_SNAPSHOT.
	dsn := "file:" + path + "?_txlock=immediate&_pragma=busy_timeout(5000)&_pragma=foreign_keys(1)"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{
		Logger:                 logger.Discard,
		SkipDefaultTransaction: true,
	})
	if err != nil {
		return nil, wrap(err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}
	sqlDB.SetMaxOpenConns(1)
	s := &Store{db: db}
	// Switching to WAL can answer BUSY at once, without waiting for
	// busy_timeout, when another process is doing the same; try again.
	err = s.prepare()
	for i := 0; i < 100 && err != nil && isBusy(err); i++ {
		time.Sleep(50 * time.Millisecond)
		err = s.prepare()
	}
	if err != nil {
		sqlDB.Close()
		return nil, err
	}
	return s, nil
}

// prepare sets WAL, checks the file and migrates it. The pragmas are the
// only SQL written by hand: they are SQLite's, not the schema's.
func (s *Store) prepare() error {
	var mode string
	if err := s.db.Raw("PRAGMA journal_mode=WAL").Row().Scan(&mode); err != nil {
		return wrap(err)
	}
	var check string
	if err := s.db.Raw("PRAGMA quick_check").Row().Scan(&check); err != nil {
		return wrap(err)
	}
	if check != "ok" {
		return &corruptError{"quick_check: " + check}
	}
	// Idempotent and inside one write transaction, so that two processes
	// opening a fresh home at once both succeed.
	return wrap(s.db.Transaction(migrate))
}

// Close closes the database.
func (s *Store) Close() error {
	sqlDB, err := s.db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

// DB is the GORM handle, for tests that look at the tables.
func (s *Store) DB() *gorm.DB { return s.db }

// Repos works outside a transaction.
func (s *Store) Repos() ports.Repos { return reposOf(s.db) }

// Write runs f in one write transaction (BEGIN IMMEDIATE).
func (s *Store) Write(f func(ports.Repos) error) error {
	return s.db.Transaction(func(tx *gorm.DB) error { return f(reposOf(tx)) })
}
