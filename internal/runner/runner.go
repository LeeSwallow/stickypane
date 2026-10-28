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

	"github.com/LeeSwallow/stickypane/internal/env"
	"github.com/LeeSwallow/stickypane/internal/store"
	"github.com/LeeSwallow/stickypane/internal/when"
)

// Run is a script that has been started.
type Run struct {
	cmd     *exec.Cmd
	log     *os.File
	started time.Time
	err     error // why it could not be run at all
	inLog   bool  // whether err is to be written into the log too
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
func Start(st *store.Store, name string, now func() time.Time) (*Run, error) {
	script := filepath.Join(st.Dir, filepath.FromSlash(name))
	log := LogOf(name)
	started := now()
	prog := "sh"
	if p, _, err := env.Detect().Script(script); err == nil {
		prog = p
	}
	head := fmt.Sprintf("$ %s %s   (%s)\n", prog, name, started.Format("2006-01-02 15:04:05"))
	if err := st.Write(log, []byte(head)); err != nil {
		return nil, err
	}

	r := &Run{started: started}
	// Not O_APPEND: on Windows that opens a handle that may only
	// append, and the sh of Git for Windows cannot write to it, so the
	// script's output would be lost. The log is the script's alone
	// while it runs, so writing on from its end is the same.
	f, err := os.OpenFile(filepath.Join(st.Dir, filepath.FromSlash(log)), os.O_WRONLY, 0o644)
	if err != nil {
		r.err = err
		return r, nil
	}
	if _, err := f.Seek(0, io.SeekEnd); err != nil {
		f.Close()
		r.err = err
		return r, nil
	}
	r.log = f
	prog, args, err := env.Detect().Script(script)
	if err == nil {
		r.cmd = exec.Command(prog, args...)
		r.cmd.Dir, r.cmd.Stdout, r.cmd.Stderr = filepath.Dir(st.Dir), f, f
		err = r.cmd.Start()
	}
	if err != nil {
		r.err, r.inLog = err, true
	}
	return r, nil
}

// Wait waits for the script to end and writes how it ended as the last
// line of its log. The error is why it could not be run at all; a script
// that ran and failed has an exit code instead.
func (r *Run) Wait() (code int, took time.Duration, err error) {
	if r.log != nil {
		defer r.log.Close()
	}
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
			fmt.Fprintf(r.log, "%v\n", err)
		}
		return 0, took, err
	}
	fmt.Fprintf(r.log, "[exit %d · %s]\n", code, when.Duration(took))
	return code, took, nil
}
