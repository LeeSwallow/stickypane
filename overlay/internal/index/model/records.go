package model

import (
	"bytes"
	"encoding/json"
	"regexp"
	"time"
)

// The records stickypane keeps in ~/.stickypane. Their form belongs to the
// main module (internal/registry, which this module cannot import) and is
// pinned there by golden files, internal/registry/testdata/home, which the
// tests of this module read too: a change to the form fails here.

// Event is one line of events/<date>.jsonl: registry.Event.
type Event struct {
	V       int    `json:"v"`
	At      string `json:"at"` // RFC 3339 (when.Moment)
	Type    string `json:"type"`
	Project string `json:"project,omitempty"`
	PID     int    `json:"pid,omitempty"`
	Kind    string `json:"kind,omitempty"`
	Cmd     string `json:"cmd,omitempty"`
}

// The event types.
const (
	ProjectInit  = "project.init"
	ProjectTouch = "project.touch"
	SessionStart = "session.start"
	SessionEnd   = "session.end"
)

// Run is the file run/<pid>.json, there while a long-lived process works
// on a project: registry.Run.
type Run struct {
	Kind    string `json:"kind"`
	PID     int    `json:"pid"`
	Project string `json:"project"`
	Started string `json:"started"` // RFC 3339 (when.Moment)
}

// ParseEvent reads one line. It is not an event (ok false) when it is not
// JSON or lacks a type, a project or a time.
func ParseEvent(line []byte) (e Event, at time.Time, ok bool) {
	if json.Unmarshal(line, &e) != nil || e.Project == "" || e.Type == "" {
		return Event{}, time.Time{}, false
	}
	at, err := time.Parse(time.RFC3339, e.At)
	if err != nil || at.IsZero() {
		return Event{}, time.Time{}, false
	}
	return e, at, true
}

// SessionKind is the kind a session.start names, tui when it names none.
func (e Event) SessionKind() string {
	if e.Kind == "" {
		return "tui"
	}
	return e.Kind
}

// Lines splits what was read from an events file into its whole lines:
// those ended by a newline. used is how many bytes they take; a last line
// without its newline is left for the next read. Blank lines are dropped.
func Lines(data []byte) (lines [][]byte, used int) {
	used = bytes.LastIndexByte(data, '\n') + 1
	if used == 0 {
		return nil, 0
	}
	for _, l := range bytes.Split(data[:used-1], []byte{'\n'}) {
		if len(bytes.TrimSpace(l)) > 0 {
			lines = append(lines, l)
		}
	}
	return lines, used
}

var runName = regexp.MustCompile(`^[0-9]+\.json$`)

// IsRunName says whether a name in run/ is a run file, not a touch marker
// or a temporary file.
func IsRunName(name string) bool { return runName.MatchString(name) }

// ParseRun reads a run file. It is not one (ok false) when it is not JSON,
// names no pid or has no start time.
func ParseRun(b []byte) (r Run, started time.Time, ok bool) {
	if json.Unmarshal(b, &r) != nil || r.PID <= 0 {
		return r, time.Time{}, false
	}
	started, err := time.Parse(time.RFC3339, r.Started)
	if err != nil {
		return r, time.Time{}, false
	}
	return r, started, true
}
