// Package registry records, on this machine, which boards are used and
// when, so that a desktop app can list them. It writes dated lines to
// events/ and one file per long-lived process to run/, inside the folder
// ~/.stickypane (or $STICKYPANE_HOME).
//
// The folder is made by the app, never here: where it does not exist, a
// person who only uses the terminal gets nothing in their home, and every
// function of this package does nothing. Writing is best-effort too: a
// command must not fail because a record could not be kept, so no function
// returns an error.
package registry

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	"github.com/LeeSwallow/stickypane/internal/when"
)

// maxLine is the longest line Append writes, newline included. Opening the
// file with O_APPEND makes each single write land at the end of the file,
// whatever other processes do; the cap keeps a line to one modest write.
// (PIPE_BUF is about pipes, not files, and is not what holds here.)
const maxLine = 4096

// Event is one line of events/<date>.jsonl. Empty fields are left out.
type Event struct {
	V       int    `json:"v"`
	At      string `json:"at"`
	Type    string `json:"type"`
	Project string `json:"project,omitempty"`
	PID     int    `json:"pid,omitempty"`
	Kind    string `json:"kind,omitempty"`
	Cmd     string `json:"cmd,omitempty"`
}

// Home is the folder the records go in: $STICKYPANE_HOME, else
// ~/.stickypane. It is named so as not to be taken for a board, which
// would be what ~/.sticky is. It is empty when no home can be found.
func Home() string {
	if h := os.Getenv("STICKYPANE_HOME"); h != "" {
		return h
	}
	h, err := os.UserHomeDir()
	if err != nil || h == "" {
		return ""
	}
	return filepath.Join(h, ".stickypane")
}

// Enabled says whether records are kept: only when the home folder
// already exists, because the app makes it.
func Enabled() bool {
	h := Home()
	if h == "" {
		return false
	}
	fi, err := os.Stat(h)
	return err == nil && fi.IsDir()
}

// Append adds e as one line to today's file in events/. It stamps the
// version and, if empty, the time. A line longer than maxLine is dropped
// rather than written in pieces another process could cut into.
func Append(e Event) {
	if !Enabled() {
		return
	}
	now := time.Now()
	e.V = 1
	if e.At == "" {
		e.At = now.Format(when.Moment)
	}
	b, err := json.Marshal(e)
	if err != nil || len(b)+1 > maxLine {
		return
	}
	dir := filepath.Join(Home(), "events")
	if os.MkdirAll(dir, 0o755) != nil {
		return
	}
	f, err := os.OpenFile(filepath.Join(dir, now.Format(when.Date)+".jsonl"),
		os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0o644)
	if err != nil {
		return
	}
	defer f.Close()
	// One Write, so that the line and its newline arrive together.
	f.Write(append(b, '\n'))
}
