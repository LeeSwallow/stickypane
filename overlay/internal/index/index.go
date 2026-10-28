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
	if _, serr := os.Stat(path); serr != nil {
		return nil, err // no file to blame
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
	db, err := sql.Open("sqlite", "file:"+path+"?_pragma=busy_timeout(5000)&_pragma=foreign_keys(1)")
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	ix := &Index{home: home, db: db, Alive: processAlive}
	if err := ix.prepare(); err != nil {
		db.Close()
		return nil, err
	}
	return ix, nil
}

func (ix *Index) prepare() error {
	var mode string
	if err := ix.db.QueryRow(`PRAGMA journal_mode=WAL`).Scan(&mode); err != nil {
		return err
	}
	var check string
	if err := ix.db.QueryRow(`PRAGMA quick_check`).Scan(&check); err != nil {
		return err
	}
	if check != "ok" {
		return errors.New("index: quick_check: " + check)
	}
	var n int
	if err := ix.db.QueryRow(`SELECT COUNT(*) FROM sqlite_master WHERE name='meta'`).Scan(&n); err != nil {
		return err
	}
	if n == 0 {
		if _, err := ix.db.Exec(schema); err != nil {
			return err
		}
		_, err := ix.db.Exec(`INSERT INTO meta VALUES ('schema_version', ?)`, strconv.Itoa(SchemaVersion))
		return err
	}
	var v string
	if err := ix.db.QueryRow(`SELECT value FROM meta WHERE key='schema_version'`).Scan(&v); err != nil {
		return err
	}
	if v != strconv.Itoa(SchemaVersion) {
		return errors.New("index: schema version " + v)
	}
	return nil
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
