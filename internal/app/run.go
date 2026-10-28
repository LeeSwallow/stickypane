package app

import (
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/LeeSwallow/stickypane/internal/when"

	tea "charm.land/bubbletea/v2"

	"github.com/LeeSwallow/stickypane/internal/runner"
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
		m.status = say(m.tr.AlreadyRunning, map[string]any{"Name": path.Base(name)})
		return
	}
	root := filepath.Dir(m.store.Dir)
	m.confirm(say(m.tr.ConfirmRun, map[string]any{"Name": path.Base(name), "Dir": filepath.Base(root)}), func() {
		m.pending = m.run(name)
	})
}

// run writes the head of a script's log and returns the command that runs
// the script in the project folder and waits for it. The script starts in
// that command, as Bubble Tea runs it, so a run is always waited for. The
// script's output goes to its log, which starts again on every run and is
// shown in the script's own pane, under the script.
func (m *Model) run(name string) tea.Cmd {
	log := runner.LogOf(name)
	r, err := runner.Prepare(m.store, name, m.now)
	if err != nil {
		m.status = say(m.tr.CannotWriteLog, map[string]any{"Log": log, "Err": err.Error()})
		return nil
	}
	// The script is opened, and its log shows in its pane. A page of a book
	// is shown by its book.
	if n := strings.Count(name, "/"); n == 0 || (n == 1 && m.store.TabOf(name) != "") {
		open := true
		_ = m.store.SetView(name, func(v *store.View) { v.Open = &open })
	}
	if m.running == nil {
		m.running = map[string]bool{}
	}
	m.running[name] = true
	m.reload()

	return func() tea.Msg {
		code, took, err := r.Wait()
		return scriptDoneMsg{name: name, code: code, err: err, took: took}
	}
}

// finished notes that a script ended and says how.
func (m *Model) finished(msg scriptDoneMsg) {
	delete(m.running, msg.name)
	switch base := path.Base(msg.name); {
	case msg.err != nil:
		m.status = say(m.tr.CouldNotRun, map[string]any{"Name": base, "Err": msg.err.Error()})
	default:
		m.status = say(m.tr.RunEnded, map[string]any{"Name": base, "Code": msg.code, "Took": when.Duration(msg.took), "Log": path.Base(runner.LogOf(msg.name))})
	}
	m.reload()
}
