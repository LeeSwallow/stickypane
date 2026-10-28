// Package index keeps the app's SQLite index of every board used on this
// machine. The files under ~/.stickypane (events/, run/) are the truth; the
// database stickypane.db is only an index of them and can be rebuilt from
// events/ at any time (docs/overlay/08 §8.4).
package index

import (
	"database/sql"
	_ "embed"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

//go:embed schema.sql
var schema string

// SchemaVersion is the version written to meta.schema_version.
const SchemaVersion = 1

// Index is the open database. Alive says whether the process pid, recorded
// as started at the given time, is still that process; tests replace it.
type Index struct {
	home  string
	db    *sql.DB
	Alive func(pid int, started time.Time) bool
}

// corruptError marks a database that must be moved aside and rebuilt: one
// that fails quick_check, is not a database file, or has another schema
// version. Every other failure (BUSY, permissions, I/O) is returned as it
// is and never costs the user the file.
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

// Open opens (making it if need be) home/stickypane.db. A database that is
// damaged, or of another schema version, is renamed to
// stickypane.db.broken-<time> and rebuilt from events/.
func Open(home string) (*Index, error) {
	if err := os.MkdirAll(home, 0o755); err != nil {
		return nil, err
	}
	path := filepath.Join(home, "stickypane.db")
	ix, err := openFile(home, path)
	if err == nil {
		return ix, nil
	}
	var ce *corruptError
	if !errors.As(err, &ce) {
		return nil, err
	}
	stamp := time.Now().Format("20060102-150405.000")
	if rerr := os.Rename(path, path+".broken-"+stamp); rerr != nil {
		return nil, fmt.Errorf("%w (and could not move it aside: %v)", err, rerr)
	}
	os.Remove(path + "-wal")
	os.Remove(path + "-shm")
	ix, err = openFile(home, path)
	if err != nil {
		return nil, err
	}
	if err := ix.Ingest(); err != nil { // offsets are empty: reads all of events/
		ix.Close()
		return nil, err
	}
	return ix, nil
}

func openFile(home, path string) (*Index, error) {
	// _txlock=immediate: every transaction takes the write lock at BEGIN, so
	// two processes never read stale state and then fail with BUSY_SNAPSHOT.
	db, err := sql.Open("sqlite", "file:"+path+"?_txlock=immediate&_pragma=busy_timeout(5000)&_pragma=foreign_keys(1)")
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	ix := &Index{home: home, db: db, Alive: processAlive}
	// Switching to WAL can answer BUSY at once, without waiting for
	// busy_timeout, when another process is doing the same; try again.
	err = ix.prepare()
	for i := 0; i < 100 && err != nil && isBusy(err); i++ {
		time.Sleep(50 * time.Millisecond)
		err = ix.prepare()
	}
	if err != nil {
		db.Close()
		return nil, err
	}
	return ix, nil
}

func (ix *Index) prepare() error {
	wrap := func(err error) error {
		if looksCorrupt(err) {
			return &corruptError{err.Error()}
		}
		return err
	}
	var mode string
	if err := ix.db.QueryRow(`PRAGMA journal_mode=WAL`).Scan(&mode); err != nil {
		return wrap(err)
	}
	var check string
	if err := ix.db.QueryRow(`PRAGMA quick_check`).Scan(&check); err != nil {
		return wrap(err)
	}
	if check != "ok" {
		return &corruptError{"quick_check: " + check}
	}
	// Idempotent and inside one write transaction, so that two processes
	// opening a fresh home at once both succeed.
	tx, err := ix.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.Exec(schema); err != nil {
		return wrap(err)
	}
	if _, err := tx.Exec(`INSERT OR IGNORE INTO meta VALUES ('schema_version', ?)`, strconv.Itoa(SchemaVersion)); err != nil {
		return wrap(err)
	}
	var v string
	if err := tx.QueryRow(`SELECT value FROM meta WHERE key='schema_version'`).Scan(&v); err != nil {
		return wrap(err)
	}
	if v != strconv.Itoa(SchemaVersion) {
		return &corruptError{"schema version " + v}
	}
	return tx.Commit()
}

// Close closes the database.
func (ix *Index) Close() error { return ix.db.Close() }

// Skipped is how many broken event lines have been skipped so far.
func (ix *Index) Skipped() int {
	var s string
	ix.db.QueryRow(`SELECT value FROM meta WHERE key='skipped_lines'`).Scan(&s)
	n, _ := strconv.Atoi(s)
	return n
}
