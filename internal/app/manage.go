package app

import (
	"os"
	"os/exec"
	"path"
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/LeeSwallow/stickypane/internal/doc"
	"github.com/LeeSwallow/stickypane/internal/store"
	"github.com/LeeSwallow/stickypane/internal/widget"
)

// sizes lists the note sizes from smallest to largest, the order "+" and
// "-" step through.
var sizes = []string{widget.SizeCard, widget.SizeHalf, widget.SizePage}

func init() {
	handlers[modeConfirm] = confirmUpdate
	footers[modeConfirm] = func(m *Model) string { return widget.Warn.Bold(true).Render(m.confirmMsg) }

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
		current := m.views[it.note.Name].Color
		if current == "" {
			current, _ = it.note.Doc.Get("color")
		}
		next := palette[(colorIndex(it.note.Name, current)+1)%len(palette)].name
		m.setView(it.note.Name, func(v *store.View) { v.Color = next })
		return nil
	})
	// The title, the archive and delete act on the file that is shown: a
	// note's own, or the page of a book.
	boardKeys["R"] = onFocused(func(m *Model, it item) tea.Cmd {
		file := it.file()
		if !strings.EqualFold(path.Ext(file.Name), ".md") {
			m.status = "Only a Markdown note has a title."
			return nil
		}
		current, _ := file.Doc.Get("title")
		m.ask("Title", current, func(title string) {
			m.apply(file.Name, doc.SetKey{Key: "title", Value: title})
		})
		return nil
	})
	boardKeys["x"] = onFocused(func(m *Model, it item) tea.Cmd {
		if err := m.store.Archive(it.file().Name); err != nil {
			m.status = "Cannot archive the note: " + err.Error()
		} else {
			m.status = "Moved to archive/."
		}
		m.reload()
		return nil
	})
	boardKeys["D"] = onFocused(func(m *Model, it item) tea.Cmd {
		file := it.file()
		m.confirm("Delete "+nameOf(file)+"? (y/n)", func() {
			if err := m.store.Delete(file.Name); err != nil {
				m.status = "Cannot delete the note: " + err.Error()
			}
			m.reload()
		})
		return nil
	})
	boardKeys["E"] = onFocused(func(_ *Model, it item) tea.Cmd {
		return tea.ExecProcess(editorCommand(it.file().Path), func(err error) tea.Msg {
			return reloadMsg{what: "Editor failed", err: err}
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
	if err := m.store.SetOrder(names); err != nil {
		m.status = "The arrangement was not saved: " + err.Error()
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

// editorCommand opens path in $EDITOR, which may carry arguments such as
// "code --wait". Without $EDITOR it falls back to vi.
func editorCommand(path string) *exec.Cmd {
	parts := strings.Fields(os.Getenv("EDITOR"))
	if len(parts) == 0 {
		parts = []string{"vi"}
	}
	return exec.Command(parts[0], append(parts[1:], path)...)
}
