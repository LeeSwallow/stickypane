package app

import (
	"context"
	"path"
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/LeeSwallow/stickypane/internal/httpfile"
	"github.com/LeeSwallow/stickypane/internal/store"
)

// httpExts are the files whose Run sends a request instead of running a
// script.
var httpExts = []string{".http", ".rest"}

// sentMsg arrives when a request of a .http file has been sent.
type sentMsg struct {
	name string // the .http file
	end  string // the last line of its log
	err  error  // why the log could not be written
}

// askSend asks whether to send a request, and sends it on a yes. The file
// is one an agent may have written, and its hooks run commands, so nothing
// is sent without the user saying so, each time.
func (m *Model) askSend(name string, part int) {
	if m.running[name] {
		m.status = say(tr.AlreadyRunning, map[string]any{"Name": path.Base(name)})
		return
	}
	data, err := m.store.Read(name)
	if err != nil {
		m.status = say(tr.CannotReadNote, map[string]any{"Err": err.Error()})
		return
	}
	f := httpfile.Parse(string(data))
	if part < 0 || part >= len(f.Requests) {
		return
	}
	r := f.Requests[part]
	what := r.Method + " " + r.URL
	if r.Name != "" {
		what = r.Name + " (" + what + ")"
	}
	var hooks []string
	for _, c := range r.Pre {
		hooks = append(hooks, "@pre "+c)
	}
	for _, c := range r.Post {
		hooks = append(hooks, "@post "+c)
	}
	m.confirm(say(tr.ConfirmSend, map[string]any{"Request": what, "Hooks": strings.Join(hooks, "; ")}), func() {
		m.pending = m.send(name, part)
	})
}

// send sends the request in the background; its log shows in the note's
// pane when it is back.
func (m *Model) send(name string, part int) tea.Cmd {
	if n := strings.Count(name, "/"); n == 0 || (n == 1 && m.store.TabOf(name) != "") {
		open := true
		_ = m.store.SetView(name, func(v *store.View) { v.Open = &open })
	}
	if m.running == nil {
		m.running = map[string]bool{}
	}
	m.running[name] = true
	m.status = say(tr.Sending, map[string]any{"Name": path.Base(name)})
	st := m.store
	return func() tea.Msg {
		res, err := httpfile.SendNote(context.Background(), st, st.Dir, name, part, "")
		return sentMsg{name: name, end: strings.Trim(httpfile.End(res), "[]"), err: err}
	}
}

// sent says how a request ended.
func (m *Model) sent(msg sentMsg) {
	delete(m.running, msg.name)
	if msg.err != nil {
		m.status = say(tr.CannotWriteLog, map[string]any{"Log": path.Base(logOf(msg.name)), "Err": msg.err.Error()})
	} else {
		m.status = msg.end
	}
	m.reload()
}
