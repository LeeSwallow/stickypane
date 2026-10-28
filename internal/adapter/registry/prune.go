package registry

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/LeeSwallow/stickypane/internal/core/when"
)

// How long records are kept. An event file is kept for keepEvents after its
// day; a touch marker, which only throttles for touchEvery, for keepMarker.
const (
	keepEvents = 90 * 24 * time.Hour
	keepMarker = 24 * time.Hour
)

// alive says whether a process with pid is running. Tests replace it, so
// that they do not depend on which pids are real.
var alive = processAlive

// Prune removes from home the records nobody needs any more: run/<pid>.json
// of a process that is no longer alive, events/<date>.jsonl of a day more
// than 90 days before now, and touch markers not touched for a day. Files
// it does not recognise are left alone. stickypane is the only program
// that deletes these files. A home that does not exist is not an error;
// the first other error is returned, after trying every file.
func Prune(home string, now time.Time) error {
	if !exists(home) {
		return nil
	}
	var first error
	remove := func(p string) {
		if err := os.Remove(p); err != nil && !errors.Is(err, fs.ErrNotExist) && first == nil {
			first = err
		}
	}
	run := filepath.Join(home, "run")
	ents, err := os.ReadDir(run)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		first = err
	}
	for _, e := range ents {
		name := e.Name()
		switch {
		case e.IsDir():
		case strings.HasPrefix(name, touchPrefix):
			if fi, err := e.Info(); err == nil && now.Sub(fi.ModTime()) > keepMarker {
				remove(filepath.Join(run, name))
			}
		case strings.HasSuffix(name, ".json"):
			pid, err := strconv.Atoi(strings.TrimSuffix(name, ".json"))
			if err == nil && pid > 0 && !alive(pid) {
				remove(filepath.Join(run, name))
			}
		}
	}
	events := filepath.Join(home, "events")
	ents, err = os.ReadDir(events)
	if err != nil && !errors.Is(err, fs.ErrNotExist) && first == nil {
		first = err
	}
	// The oldest day kept, as a date in now's zone: dates compare as text.
	oldest := now.Add(-keepEvents).Format(when.Date)
	for _, e := range ents {
		day, ok := strings.CutSuffix(e.Name(), ".jsonl")
		if e.IsDir() || !ok {
			continue
		}
		if _, err := time.Parse(when.Date, day); err == nil && day < oldest {
			remove(filepath.Join(events, e.Name()))
		}
	}
	return first
}
