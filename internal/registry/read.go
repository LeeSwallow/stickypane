package registry

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

// Read reads the records in home, for the desktop app: the run files, by
// pid, and every event, oldest day first and in the order written within a
// day. A file or line that is not whole (one being written, or cut short)
// is skipped. Whether a run's process is still alive is not checked;
// Prune removes the files of those that are not. A home that does not
// exist holds nothing, and is not an error.
func Read(home string) (runs []Run, events []Event, err error) {
	if !exists(home) {
		return nil, nil, nil
	}
	run := filepath.Join(home, "run")
	ents, err := os.ReadDir(run)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return nil, nil, err
	}
	for _, e := range ents {
		name, ok := strings.CutSuffix(e.Name(), ".json")
		if e.IsDir() || !ok {
			continue
		}
		if _, err := strconv.Atoi(name); err != nil {
			continue
		}
		b, err := os.ReadFile(filepath.Join(run, e.Name()))
		var r Run
		if err != nil || json.Unmarshal(b, &r) != nil {
			continue
		}
		runs = append(runs, r)
	}
	sort.Slice(runs, func(i, j int) bool { return runs[i].PID < runs[j].PID })

	dir := filepath.Join(home, "events")
	ents, err = os.ReadDir(dir) // sorted by name, so by day
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return nil, nil, err
	}
	for _, e := range ents {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".jsonl") {
			continue
		}
		b, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			continue
		}
		sc := bufio.NewScanner(bytes.NewReader(b))
		sc.Buffer(make([]byte, 0, maxLine), 16*maxLine)
		for sc.Scan() {
			var ev Event
			if json.Unmarshal(sc.Bytes(), &ev) == nil {
				events = append(events, ev)
			}
		}
	}
	return runs, events, nil
}
