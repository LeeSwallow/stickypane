package app

import (
	"os/exec"
	"path"
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/LeeSwallow/stickypane/internal/arrange"
	"github.com/LeeSwallow/stickypane/internal/doc"
	"github.com/LeeSwallow/stickypane/internal/env"
	"github.com/LeeSwallow/stickypane/internal/store"
	"github.com/LeeSwallow/stickypane/internal/widget"
)

// sizes lists the note sizes from smallest to largest, the order "+" and
// "-" step through.
var sizes = []string{widget.SizeCard, widget.SizeHalf, widget.SizePage}

func init() {
	handlers[modeConfirm] = confirmUpdate
	footers[modeConfirm] = confirmFooter

	grow := onFocused(func(m *Model, it item) tea.Cmd { m.resize(it, 1); return nil })
	boardKeys["+"] = grow
	boardKeys["="] = grow // the same key without shift
	boardKeys["-"] = onFocused(func(m *Model, it item) tea.Cmd { m.resize(it, -1); return nil })
	boardKeys["}"] = func(m *Model) tea.Cmd { m.move(1); return nil }
	boardKeys["{"] = func(m *Model) tea.Cmd { m.move(-1); return nil }

	// Pin and color are part of how the screen is arranged: they go to
	// sticky.json, like open and size.
	boardKeys["p"] = onFocused(func(m *Model, it item) tea.Cmd {
		pin := !m.pinned(it)
		m.setView(it.note.Name, func(v *store.View) { v.Pin = &pin })
		return nil
	})
	boardKeys["c"] = onFocused(func(m *Model, it item) tea.Cmd {
		current := arrange.Color(m.views[it.note.Name], it.note.Doc)
		next := palette[(colorIndex(it.note.Name, current)+1)%len(palette)]
		m.setView(it.note.Name, func(v *store.View) { v.Color = next })
		return nil
	})
	// The title, the archive and delete act on the file that is shown: a
	// note's own, or the page of a book.
	// A Markdown note keeps its title in its front matter. A book, a log
	// or a script has none, so its name is kept in sticky.json; the file or
	// folder is not renamed either way.
	boardKeys["R"] = onFocused(func(m *Model, it item) tea.Cmd {
		file := it.file()
		if len(it.pages) > 0 || !strings.EqualFold(path.Ext(file.Name), ".md") || m.store.Linked(file.Name) {
			name := it.note.Name
			m.ask(tr.Name, m.views[name].Title, func(title string) {
				m.setView(name, func(v *store.View) { v.Title = title })
			})
			return nil
		}
		current, _ := file.Doc.Get("title")
		m.ask(tr.Title, current, func(title string) {
			m.apply(file.Name, doc.SetKey{Key: "title", Value: title})
		})
		return nil
	})
	// Neither x nor D removes anything from the disk: the file goes to
	// archive/ or to .trash/, and u brings it back.
	boardKeys["x"] = onFocused(func(m *Model, it item) tea.Cmd {
		file := it.file()
		to, err := m.store.Archive(file.Name)
		if err != nil {
			m.status = say(tr.CannotArchive, map[string]any{"Err": err.Error()})
		} else {
			m.undo = &undo{from: to, to: file.Name, label: nameOf(file)}
			m.status = say(tr.Archived, map[string]any{"Name": nameOf(file)})
		}
		m.reload()
		return nil
	})
	boardKeys["D"] = onFocused(func(m *Model, it item) tea.Cmd {
		file := it.file()
		m.confirm(say(tr.ConfirmDelete, map[string]any{"Name": nameOf(file)}), func() {
			to, err := m.store.Trash(file.Name)
			if err != nil {
				m.status = say(tr.CannotDelete, map[string]any{"Err": err.Error()})
			} else {
				m.undo = &undo{from: to, to: file.Name, label: nameOf(file)}
				m.status = say(tr.Deleted, map[string]any{"Name": nameOf(file)})
			}
			m.reload()
		})
		return nil
	})
	boardKeys["E"] = onFocused(func(_ *Model, it item) tea.Cmd {
		return tea.ExecProcess(editorCommand(it.file().Path), func(err error) tea.Msg {
			return reloadMsg{what: tr.EditorFailed, err: err}
		})
	})
}

// resize steps an open note's size up or down and writes it. It writes
// nothing where nothing would show: on a closed note, and at either end of
// the sizes.
func (m *Model) resize(it item, delta int) {
	if !m.isOpen(it) {
		return
	}
	current := m.sizeOf(it)
	i := 0
	for j, s := range sizes {
		if s == current {
			i = j
		}
	}
	next := sizes[max(min(i+delta, len(sizes)-1), 0)]
	if next == current {
		return
	}
	m.setView(it.note.Name, func(v *store.View) { v.Size = next })
	m.reveal = revealNote
}

// move swaps the focused note with its neighbor in the order of the notes,
// which is the order of the title bar and of the panes. A pinned note moves
// among the pinned ones, the others among themselves.
func (m *Model) move(delta int) {
	i := m.index(m.focus)
	j := i + delta
	if i < 0 || j < 0 || j >= len(m.items) || m.pinned(m.items[i]) != m.pinned(m.items[j]) {
		return
	}
	names := make([]string, len(m.items))
	for k, it := range m.items {
		names[k] = it.note.Name
	}
	names[i], names[j] = names[j], names[i]
	if err := m.store.SetOrder(m.tabName, names); err != nil {
		m.status = say(tr.ArrangementNotSaved, map[string]any{"Err": err.Error()})
	}
	m.reload()
	m.reveal = revealNote
}

// confirm asks a yes/no question on the bottom line.
func (m *Model) confirm(msg string, yes func()) {
	m.confirmMsg, m.onConfirm = msg, yes
	m.back, m.mode = m.mode, modeConfirm
}

func confirmUpdate(m *Model, msg tea.Msg) tea.Cmd {
	k, ok := msg.(tea.KeyPressMsg)
	if !ok {
		return nil
	}
	m.mode = m.back
	if s := k.String(); s == "y" || s == "Y" {
		m.onConfirm()
	}
	return nil
}

// editorCommand opens path in the user's editor, any of them: a GUI editor
// is told to wait until the file is closed (see env.EditorCommand).
func editorCommand(path string) *exec.Cmd {
	cmd := env.Detect().WithEditor(editorChoice).EditorCommand(path)
	return exec.Command(cmd[0], cmd[1:]...)
}

// confirmFooter is the question with its answers as buttons, so that a
// question can be answered with the mouse as well as with y and n. It
// keeps where the buttons are for click.
func confirmFooter(m *Model) string {
	q := widget.Warn.Bold(true).Render(m.confirmMsg) + "  "
	yes, no := "[ "+widget.T("Yes")+" ]", "[ "+widget.T("No")+" ]"
	x := widget.Width(q)
	m.confirmYes = [2]int{x, x + widget.Width(yes)}
	x += widget.Width(yes) + 1
	m.confirmNo = [2]int{x, x + widget.Width(no)}
	return q + widget.Accent.Bold(true).Render(yes) + " " + widget.Faint.Render(no)
}
