// Package registry records, on this machine, which boards are used and
// when, so that a desktop app can list them. It writes dated lines to
// events/ and one file per long-lived process to run/, inside the folder
// ~/.stickypane (or $STICKYPANE_HOME).
//
// Records are kept only where the desktop app has written its settings
// file, AppFile, in that folder; neither is made here. A person who only
// uses the terminal gets nothing recorded, even with the folder there for
// something else, and every writer of this package does nothing. Writing is best-effort too: a
// command must not fail because a record could not be kept, so no writer
// returns an error.
//
// stickypane is the only program that writes or deletes these files; the
// app reads them, with Read. The files' form is pinned by testdata/home.
package registry

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	"github.com/LeeSwallow/stickypane/internal/adapter/atomicfile"
	"github.com/LeeSwallow/stickypane/internal/core/when"
)

// maxLine is the longest line Append writes, newline included. Opening the
// file with O_APPEND makes each single write land at the end of the file,
// whatever other processes do; the cap keeps a line to one modest write.
// (PIPE_BUF is about pipes, not files, and is not what holds here.)
const maxLine = 4096

// Where the records are kept, inside the home folder: a run file is
// run/<pid>.json, a day of events events/<date>.jsonl.
const (
	// HomeVar names the home folder in place of ~/.stickypane.
	HomeVar  = "STICKYPANE_HOME"
	homeName = ".stickypane"
	runDir   = "run"
	runExt   = ".json"
	eventDir = "events"
	eventExt = ".jsonl"
	// AppFile is the desktop app's settings file in the home folder
	// (desktop/internal/shell.StateFile, in the app's own module). The app
	// writes it on start; that it is there is what turns records on.
	AppFile = "overlay.json"
)

// The types of an Event.
const (
	ProjectInit  = "project.init"
	ProjectTouch = "project.touch"
	SessionStart = "session.start"
	SessionEnd   = "session.end"
)

// The kinds of a long-lived process, in a Run and a session.start.
const (
	KindTUI    = "tui"
	KindBridge = "bridge"
	KindMCP    = "mcp"
	KindWatch  = "watch"
)

// Event is one line of events/<date>.jsonl, the date a when.Date. Empty
// fields are left out. At is a when.Moment. Type is project.init,
// project.touch (with Cmd), session.start (with PID and Kind) or
// session.end (with PID).
type Event struct {
	V       int    `json:"v"`
	At      string `json:"at"`
	Type    string `json:"type"`
	Project string `json:"project,omitempty"`
	PID     int    `json:"pid,omitempty"`
	Kind    string `json:"kind,omitempty"`
	Cmd     string `json:"cmd,omitempty"`
}

// Run is the file run/<pid>.json, there while a long-lived process works
// on a project. Kind is tui, bridge, mcp or watch; Started is a
// when.Moment. The fields are in the order of their names, as the file was
// first written from a map.
type Run struct {
	Kind    string `json:"kind"`
	PID     int    `json:"pid"`
	Project string `json:"project"`
	Started string `json:"started"`
}

// Writer keeps the records of one process in the folder Home. Now is its
// clock and PID the process it speaks for; tests set both.
type Writer struct {
	Home string
	Now  func() time.Time
	PID  int
}

// Default is the writer of this process: in Home(), on the real clock.
func Default() Writer {
	return Writer{Home: Home(), Now: time.Now, PID: os.Getpid()}
}

// Home is the folder the records go in: $STICKYPANE_HOME, else
// ~/.stickypane. It is named so as not to be taken for a board, which
// would be what ~/.sticky is. It is empty when no home can be found.
func Home() string {
	if h := os.Getenv(HomeVar); h != "" {
		return h
	}
	h, err := os.UserHomeDir()
	if err != nil || h == "" {
		return ""
	}
	return filepath.Join(h, homeName)
}

// Enabled says whether records are kept: only when the app's AppFile is
// in the home folder, because the app writes it.
func Enabled() bool { return enabled(Home()) }

// enabled says whether records are kept in home: AppFile is there.
func enabled(home string) bool {
	if home == "" {
		return false
	}
	fi, err := os.Stat(filepath.Join(home, AppFile))
	return err == nil && fi.Mode().IsRegular()
}

func exists(home string) bool {
	if home == "" {
		return false
	}
	fi, err := os.Stat(home)
	return err == nil && fi.IsDir()
}

func (w Writer) now() time.Time {
	if w.Now == nil {
		return time.Now()
	}
	return w.Now()
}

// Append adds e to today's file through Default.
func Append(e Event) { Default().Append(e) }

// Append adds e as one line to today's file in events/. It stamps the
// version and, if empty, the time. A line longer than maxLine is dropped
// rather than written in pieces another process could cut into.
func (w Writer) Append(e Event) {
	if !enabled(w.Home) {
		return
	}
	now := w.now()
	e.V = 1
	if e.At == "" {
		e.At = now.Format(when.Moment)
	}
	b, err := json.Marshal(e)
	if err != nil || len(b)+1 > maxLine {
		return
	}
	dir := filepath.Join(w.Home, eventDir)
	if os.MkdirAll(dir, atomicfile.DirMode) != nil {
		return
	}
	f, err := os.OpenFile(filepath.Join(dir, now.Format(when.Date)+eventExt),
		os.O_WRONLY|os.O_APPEND|os.O_CREATE, atomicfile.FileMode)
	if err != nil {
		return
	}
	defer f.Close()
	// One Write, so that the line and its newline arrive together.
	f.Write(append(b, '\n'))
}
