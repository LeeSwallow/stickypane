package app

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/LeeSwallow/stickypane/internal/env"
	"github.com/LeeSwallow/stickypane/internal/store"
)

// scriptDoneMsg arrives when a script has ended.
type scriptDoneMsg struct {
	name string // the script's file name
	code int    // its exit code
	err  error  // why it could not be run at all
	took time.Duration
}

// askRun asks whether to run a script, and starts it on a yes. A script is
// a file an agent may have written, so it never runs without the user
// saying so, each time.
func (m *Model) askRun(name string) {
	if m.running[name] {
		m.status = say(tr.AlreadyRunning, map[string]any{"Name": path.Base(name)})
		return
	}
	root := filepath.Dir(m.store.Dir)
	m.confirm(say(tr.ConfirmRun, map[string]any{"Name": path.Base(name), "Dir": filepath.Base(root)}), func() {
		m.pending = m.run(name)
	})
}

// logOf names the log a script writes to: the same name with ".log", next
// to it. In a book that makes the log another page of the book.
func logOf(name string) string {
	return strings.TrimSuffix(name, path.Ext(name)) + ".log"
}

// run starts a script in the project folder and returns the command that
// waits for it. The script's output goes to its log, which starts again on
// every run and is opened on the screen, where it is followed like any log.
func (m *Model) run(name string) tea.Cmd {
	script := filepath.Join(m.store.Dir, filepath.FromSlash(name))
	log := logOf(name)
	started := m.now()
	head := fmt.Sprintf("$ sh %s   (%s)\n", name, started.Format("2006-01-02 15:04:05"))
	if err := m.store.Write(log, []byte(head)); err != nil {
		m.status = say(tr.CannotWriteLog, map[string]any{"Log": log, "Err": err.Error()})
		return nil
	}
	// A note of the root or of a tab is opened; a page of a book is shown
	// by its book.
	if n := strings.Count(log, "/"); n == 0 || (n == 1 && m.store.TabOf(log) != "") {
		open := true
		_ = m.store.SetView(log, func(v *store.View) { v.Open = &open })
	}
	if m.running == nil {
		m.running = map[string]bool{}
	}
	m.running[name] = true
	m.reload()

	root := filepath.Dir(m.store.Dir)
	out := filepath.Join(m.store.Dir, filepath.FromSlash(log))
	return func() tea.Msg {
		done := scriptDoneMsg{name: name}
		f, err := os.OpenFile(out, os.O_APPEND|os.O_WRONLY, 0o644)
		if err != nil {
			done.err = err
			return done
		}
		defer f.Close()
		prog, args, err := env.Detect().Script(script)
		if err == nil {
			cmd := exec.Command(prog, args...)
			cmd.Dir, cmd.Stdout, cmd.Stderr = root, f, f
			err = cmd.Run()
		}
		done.took = time.Since(started)
		var exit *exec.ExitError
		switch {
		case errors.As(err, &exit):
			done.code = exit.ExitCode()
		case err != nil:
			done.err = err
			fmt.Fprintf(f, "%v\n", err)
			return done
		}
		fmt.Fprintf(f, "[exit %d · %s]\n", done.code, done.took.Round(10*time.Millisecond))
		return done
	}
}

// finished notes that a script ended and says how.
func (m *Model) finished(msg scriptDoneMsg) {
	delete(m.running, msg.name)
	switch base := path.Base(msg.name); {
	case msg.err != nil:
		m.status = say(tr.CouldNotRun, map[string]any{"Name": base, "Err": msg.err.Error()})
	default:
		m.status = say(tr.RunEnded, map[string]any{"Name": base, "Code": msg.code, "Took": msg.took.Round(10 * time.Millisecond).String(), "Log": path.Base(logOf(msg.name))})
	}
	m.reload()
}
