package app

import (
	"errors"
	"path"
	"slices"

	"github.com/LeeSwallow/stickypane/internal/doc"
	"github.com/LeeSwallow/stickypane/internal/widget"
)

// apply writes an intent to the file called name and refreshes the screen. On a conflict
// nothing is written and the screen simply catches up with the file.
func (m *Model) apply(name string, op doc.Op) {
	err := m.store.Apply(name, op)
	switch {
	case errors.Is(err, doc.ErrConflict):
		m.status = tr.Conflict
	case err != nil:
		m.status = say(tr.WriteFailed, map[string]any{"Err": err.Error()})
	}
	m.reload()
	i := m.showing(name)
	if i < 0 {
		return
	}
	if err != nil {
		// The widget may already show the change that did not happen.
		// Rebuild it from the file so the screen never claims otherwise.
		m.setWidget(i, m.items[i].w.Sync(m.items[i].file().Doc))
	}
	m.markSeen(m.items[i].note.Name)
}

// toWidget sends a key to a note's widget and carries out what it asks for.
func (m *Model) toWidget(i int, key string) {
	name := m.items[i].file().Name
	w, res := m.items[i].w.Update(key)
	m.setWidget(i, w)
	m.reveal = revealCursor
	m.act(name, res)
}

// act carries out what a widget asked for.
func (m *Model) act(name string, res widget.Result) {
	if res.Op != nil {
		m.apply(name, res.Op)
	}
	if res.Run {
		if slices.Contains(httpExts, path.Ext(name)) {
			m.askSend(name, res.Part)
		} else {
			m.askRun(name)
		}
	}
	if p := res.Prompt; p != nil {
		m.ask(p.Label, p.Initial, func(text string) {
			m.apply(name, p.Submit(text))
			m.reveal = revealCursor
		})
		m.inputEmpty = p.Empty
	}
}
