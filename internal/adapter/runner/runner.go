// Package runner runs a script of the board and keeps its log. It sits
// below the screens so that every front end runs a script the same way.
package runner

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/LeeSwallow/stickypane/internal/adapter/env"
	"github.com/LeeSwallow/stickypane/internal/adapter/store"
	"github.com/LeeSwallow/stickypane/internal/core/when"
)

// Run is a script to run: prepared, started, or ended.
type Run struct {
	script   string // the script's path
	dir      string // the project folder, where it runs
	log      string // the log's path
	cmd      *exec.Cmd
	started  time.Time
	launched bool  // whether launch has been called
	err      error // why it could not be run at all
	inLog    bool  // whether err is to be written into the log too
	group    bool  // whether it leads a process group of its own
}

// LogOf names the log a script writes to: the same name with ".log", next
// to it. In a book that makes the log another page of the book.
func LogOf(name string) string {
	return strings.TrimSuffix(name, path.Ext(name)) + ".log"
}

// Start begins a script in the project folder. Its output goes to its log,
// which starts again on every run and is shown in the script's own pane,
// under the script. An error means the log could not be written; a script
// that cannot be run is reported by Wait. Opening the script's pane is the
// caller's to do.
//
// The script stays in the caller's process group, as a program started
// from a terminal does: it hangs up with the terminal and may read it.
func Start(st *store.Store, name string, now func() time.Time) (*Run, error) {
	r, err := Prepare(st, name, now)
	if err == nil {
		r.launch()
	}
	return r, err
}

// StartGroup is Start for a front end with no terminal, the desktop
// overlay: on unix the script leads a process group of its own, so that
// Kill stops what it started too (a server, a sleep).
func StartGroup(st *store.Store, name string, now func() time.Time) (*Run, error) {
	r, err := Prepare(st, name, now)
	if err == nil {
		r.group = true
		r.launch()
	}
	return r, err
}

// Prepare starts a script's log again with its head line, and starts
// nothing: Wait starts the script, then waits for it. The terminal writes
// the head at once and runs the script in a command of its own, so that a
// run nobody waits for never starts.
func Prepare(st *store.Store, name string, now func() time.Time) (*Run, error) {
	script := filepath.Join(st.Dir, filepath.FromSlash(name))
	log := LogOf(name)
	started := now()
	prog := "sh"
	if p, _, err := env.Detect().Script(script); err == nil {
		prog = p
	}
	head := fmt.Sprintf("$ %s %s   (%s)\n", prog, name, when.LogHead(started))
	if err := st.Write(log, []byte(head)); err != nil {
		return nil, err
	}
	return &Run{
		script:  script,
		dir:     filepath.Dir(st.Dir),
		log:     filepath.Join(st.Dir, filepath.FromSlash(log)),
		started: started,
	}, nil
}

// launch starts the script with its log as its output. The board closes
// its own handle on the log as soon as the script has one: the script is
// then the only one holding the log, and on Windows the log can be moved
// or removed once the script has ended.
func (r *Run) launch() {
	if r.launched {
		return
	}
	r.launched = true
	f, err := openLog(r.log)
	if err != nil {
		r.err = err
		return
	}
	defer f.Close()
	prog, args, err := env.Detect().Script(r.script)
	if err == nil {
		r.cmd = exec.Command(prog, args...)
		if r.group {
			inOwnGroup(r.cmd)
		}
		r.cmd.Dir, r.cmd.Stdout, r.cmd.Stderr = r.dir, f, f
		err = r.cmd.Start()
	}
	if err != nil {
		r.cmd, r.err, r.inLog = nil, err, true
	}
}

// openLog opens a log to write on from its end. Not O_APPEND: on Windows
// that opens a handle that may only append, and the sh of Git for Windows
// cannot write to it, so the script's output would be lost. The log is
// the script's alone while it runs, so writing on from its end is the
// same.
func openLog(name string) (*os.File, error) {
	f, err := os.OpenFile(name, os.O_WRONLY, 0o644)
	if err != nil {
		return nil, err
	}
	if _, err := f.Seek(0, io.SeekEnd); err != nil {
		f.Close()
		return nil, err
	}
	return f, nil
}

// addToLog writes a line at the end of the log, and lets go of it.
func (r *Run) addToLog(format string, a ...any) {
	f, err := openLog(r.log)
	if err != nil {
		return
	}
	fmt.Fprintf(f, format, a...)
	f.Close()
}

// Kill stops the script, for a front end that is ending while it still
// runs. A script begun by StartGroup is stopped with everything it started
// (on unix; on Windows only its own process). It is harmless on a script
// that has ended or that could not be run, and on a second call.
func (r *Run) Kill() {
	if r.cmd == nil || r.cmd.Process == nil {
		return
	}
	if r.group {
		killAll(r.cmd.Process)
		return
	}
	_ = r.cmd.Process.Kill()
}

// Wait starts a prepared script, waits for it to end and writes how it
// ended as the last line of its log. The error is why it could not be run at all; a script
// that ran and failed has an exit code instead.
func (r *Run) Wait() (code int, took time.Duration, err error) {
	r.launch()
	err = r.err
	if err == nil {
		err = r.cmd.Wait()
	}
	took = time.Since(r.started)
	var exit *exec.ExitError
	switch {
	case errors.As(err, &exit):
		code = exit.ExitCode()
	case err != nil:
		if r.inLog {
			r.addToLog("%v\n", err)
		}
		return 0, took, err
	}
	r.addToLog("[exit %d · %s]\n", code, when.Duration(took))
	return code, took, nil
}
