package index

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// event is one line of events/YYYY-MM-DD.jsonl (08 §8.4.2).
type event struct {
	V       int    `json:"v"`
	At      string `json:"at"`
	Type    string `json:"type"`
	Project string `json:"project"`
	PID     int    `json:"pid"`
	Kind    string `json:"kind"`
}

// Ingest applies the lines of events/ not yet read, then settles run/: the
// run file of a process that is no longer the one it names is removed and
// its sessions ended. A last line without its newline is left for next time;
// a line that is not an event is skipped and counted (Skipped).
func (ix *Index) Ingest() error {
	files, _ := filepath.Glob(filepath.Join(ix.home, "events", "*.jsonl"))
	present := map[string]bool{}
	for _, f := range files {
		name := filepath.Base(f)
		present[name] = true
		if err := ix.ingestFile(f, name); err != nil {
			return err
		}
	}
	rows, err := ix.db.Query(`SELECT file FROM ingest`)
	if err != nil {
		return err
	}
	var gone []string
	for rows.Next() {
		var f string
		if err := rows.Scan(&f); err != nil {
			rows.Close()
			return err
		}
		if !present[f] {
			gone = append(gone, f)
		}
	}
	rows.Close()
	for _, f := range gone { // what was applied stays; only the offset goes
		if _, err := ix.db.Exec(`DELETE FROM ingest WHERE file=?`, f); err != nil {
			return err
		}
	}
	return ix.settleRuns()
}

func (ix *Index) ingestFile(path, name string) error {
	// The offset is read inside the write transaction (BEGIN IMMEDIATE), so
	// two instances never apply the same lines twice.
	tx, err := ix.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var off int64
	err = tx.QueryRow(`SELECT offset FROM ingest WHERE file=?`, name).Scan(&off)
	if err != nil && err != sql.ErrNoRows {
		return err
	}
	f, err := os.Open(path)
	if err != nil {
		return nil // removed since the listing
	}
	defer f.Close()
	// Known limit: a file truncated and then grown past its old offset is
	// not noticed, and the bytes before the offset are never re-read.
	if fi, err := f.Stat(); err == nil && fi.Size() < off {
		off = 0 // a different, shorter file under the same name
	}
	if _, err := f.Seek(off, io.SeekStart); err != nil {
		return err
	}
	data, err := io.ReadAll(f)
	if err != nil {
		return err
	}
	end := bytes.LastIndexByte(data, '\n') + 1
	if end == 0 {
		return nil
	}
	skipped := 0
	for _, line := range bytes.Split(data[:end-1], []byte{'\n'}) {
		if len(bytes.TrimSpace(line)) == 0 {
			continue
		}
		var e event
		at, perr := time.Time{}, error(nil)
		if json.Unmarshal(line, &e) == nil {
			at, perr = time.Parse(time.RFC3339, e.At)
		}
		if e.Project == "" || e.Type == "" || perr != nil || at.IsZero() {
			skipped++
			continue
		}
		if err := applyEvent(tx, e, at); err != nil {
			return err
		}
	}
	if skipped > 0 {
		if _, err := tx.Exec(`INSERT INTO meta VALUES ('skipped_lines', ?)
			ON CONFLICT(key) DO UPDATE SET value = CAST(value AS INTEGER) + ?`, strconv.Itoa(skipped), skipped); err != nil {
			return err
		}
	}
	if _, err := tx.Exec(`INSERT INTO ingest VALUES (?, ?) ON CONFLICT(file) DO UPDATE SET offset=excluded.offset`,
		name, off+int64(end)); err != nil {
		return err
	}
	return tx.Commit()
}

func stamp(t time.Time) string { return t.Format(time.RFC3339) }

func applyEvent(tx *sql.Tx, e event, at time.Time) error {
	path := filepath.Clean(e.Project)
	var id int64
	var lastSeen string
	var removed sql.NullString
	err := tx.QueryRow(`SELECT id, last_seen, removed_at FROM project WHERE path=?`, path).Scan(&id, &lastSeen, &removed)
	switch {
	case err == sql.ErrNoRows:
		res, err := tx.Exec(`INSERT INTO project (path, title, first_seen, last_seen) VALUES (?, ?, ?, ?)`,
			path, projectTitle(path), stamp(at), stamp(at))
		if err != nil {
			return err
		}
		if id, err = res.LastInsertId(); err != nil {
			return err
		}
	case err != nil:
		return err
	default:
		if t, err := time.Parse(time.RFC3339, lastSeen); err != nil || at.After(t) {
			if _, err := tx.Exec(`UPDATE project SET last_seen=? WHERE id=?`, stamp(at), id); err != nil {
				return err
			}
		}
		if removed.Valid && boardExists(path) { // an event for a folder that is back
			if _, err := tx.Exec(`UPDATE project SET removed_at=NULL WHERE id=?`, id); err != nil {
				return err
			}
		}
	}

	// Known limit: the day is the local date of the event time in the zone
	// in force now; an event written in the repeated hour of a DST change
	// can fall on the neighbouring date.
	date := at.In(time.Local).Format("2006-01-02")
	var n int
	var first, last string
	err = tx.QueryRow(`SELECT events, first_at, last_at FROM day WHERE project_id=? AND date=?`, id, date).Scan(&n, &first, &last)
	if err == sql.ErrNoRows {
		if _, err := tx.Exec(`INSERT INTO day VALUES (?, ?, 1, ?, ?)`, id, date, stamp(at), stamp(at)); err != nil {
			return err
		}
	} else if err != nil {
		return err
	} else {
		if t, err := time.Parse(time.RFC3339, first); err != nil || at.Before(t) {
			first = stamp(at)
		}
		if t, err := time.Parse(time.RFC3339, last); err != nil || at.After(t) {
			last = stamp(at)
		}
		if _, err := tx.Exec(`UPDATE day SET events=?, first_at=?, last_at=? WHERE project_id=? AND date=?`,
			n+1, first, last, id, date); err != nil {
			return err
		}
	}

	err = nil
	switch e.Type {
	case "session.start":
		if e.PID > 0 {
			kind := e.Kind
			if kind == "" {
				kind = "tui"
			}
			_, err = tx.Exec(`INSERT OR IGNORE INTO session (pid, project_id, kind, started_at) VALUES (?, ?, ?, ?)`,
				e.PID, id, kind, stamp(at))
		}
	case "session.end":
		if e.PID > 0 {
			_, err = tx.Exec(`UPDATE session SET ended_at=? WHERE pid=? AND project_id=? AND ended_at IS NULL`,
				stamp(at), e.PID, id)
		}
	}
	return err
}

// runFile is run/<pid>.json.
type runFile struct {
	PID     int    `json:"pid"`
	Project string `json:"project"`
	Kind    string `json:"kind"`
	Started string `json:"started"`
}

var runName = regexp.MustCompile(`^[0-9]+\.json$`)

// liveRuns returns the run files whose process is alive. With reap, the
// files of the others are removed and their sessions ended.
func (ix *Index) liveRuns(reap bool) ([]runFile, error) {
	entries, err := os.ReadDir(filepath.Join(ix.home, "run"))
	if err != nil {
		return nil, nil
	}
	var live []runFile
	for _, e := range entries {
		if !runName.MatchString(e.Name()) { // touch-* markers, .tmp-* files
			continue
		}
		file := filepath.Join(ix.home, "run", e.Name())
		b, err := os.ReadFile(file)
		if err != nil {
			continue
		}
		var r runFile
		started, perr := time.Time{}, error(nil)
		if json.Unmarshal(b, &r) == nil {
			started, perr = time.Parse(time.RFC3339, r.Started)
		}
		if r.PID > 0 && perr == nil && ix.Alive(r.PID, started) {
			live = append(live, r)
			continue
		}
		if !reap {
			continue
		}
		if r.PID > 0 {
			if _, err := ix.db.Exec(`UPDATE session SET ended_at=? WHERE pid=? AND ended_at IS NULL`,
				stamp(time.Now()), r.PID); err != nil {
				return nil, err
			}
		}
		os.Remove(file)
	}
	return live, nil
}

func (ix *Index) settleRuns() error {
	_, err := ix.liveRuns(true)
	return err
}

// processAlive is the default liveness check: the process exists and was
// started when the run file says (within a few seconds, since the file is
// written just after the process begins). A pid reused by another program
// started later, so it does not match.
func processAlive(pid int, started time.Time) bool {
	cmd := exec.Command("ps", "-o", "lstart=", "-p", strconv.Itoa(pid))
	cmd.Env = append(os.Environ(), "LC_ALL=C", "LANG=C")
	out, err := cmd.Output()
	if err != nil {
		return false
	}
	s := strings.Join(strings.Fields(string(out)), " ")
	if s == "" {
		return false
	}
	t, err := time.ParseInLocation("Mon Jan 2 15:04:05 2006", s, time.Local)
	if err != nil {
		return false
	}
	d := started.Sub(t)
	return d > -10*time.Second && d < 10*time.Second
}
