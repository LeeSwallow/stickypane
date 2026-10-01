package app

import (
	"os"
	"os/exec"
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/LeeSwallow/stickypane/internal/doc"
	"github.com/LeeSwallow/stickypane/internal/widget"
)

// sizes lists the note sizes from smallest to largest, the order "+" and
// "-" step through.
var sizes = []string{widget.SizeCard, widget.SizeHalf, widget.SizePage}

func init() {
	handlers[modeConfirm] = confirmUpdate
	footers[modeConfirm] = func(m *Model) string { return m.confirmMsg }

	grow := onFocused(func(m *Model, it item) tea.Cmd { m.resize(it, 1); return nil })
	boardKeys["+"] = grow
	boardKeys["="] = grow // the same key without shift
	boardKeys["-"] = onFocused(func(m *Model, it item) tea.Cmd { m.resize(it, -1); return nil })

	boardKeys["p"] = onFocused(func(m *Model, it item) tea.Cmd {
		value := "true"
		if it.note.Doc.Pinned() {
			value = "false"
		}
		m.apply(it.note.Name, doc.SetKey{Key: "pin", Value: value})
		return nil
	})
	boardKeys["c"] = onFocused(func(m *Model, it item) tea.Cmd {
		current, _ := it.note.Doc.Get("color")
		next := palette[(colorIndex(it.note.Name, current)+1)%len(palette)].name
		m.apply(it.note.Name, doc.SetKey{Key: "color", Value: next})
		return nil
	})
	boardKeys["R"] = onFocused(func(m *Model, it item) tea.Cmd {
		name := it.note.Name
		current, _ := it.note.Doc.Get("title")
		m.ask("Title", current, func(title string) {
			m.apply(name, doc.SetKey{Key: "title", Value: title})
		})
		return nil
	})
	boardKeys["x"] = onFocused(func(m *Model, it item) tea.Cmd {
		if err := m.store.Archive(it.note.Name); err != nil {
			m.status = "Cannot archive the note: " + err.Error()
		} else {
			m.status = "Moved to archive/."
		}
		m.reload()
		return nil
	})
	boardKeys["D"] = onFocused(func(m *Model, it item) tea.Cmd {
		name := it.note.Name
		m.confirm("Delete "+label(it)+"? (y/n)", func() {
			if err := m.store.Delete(name); err != nil {
				m.status = "Cannot delete the note: " + err.Error()
			}
			m.reload()
		})
		return nil
	})
	boardKeys["e"] = onFocused(func(_ *Model, it item) tea.Cmd {
		return tea.ExecProcess(editorCommand(it.note.Path), func(err error) tea.Msg {
			return reloadMsg{what: "Editor failed", err: err}
		})
	})
}

// resize steps a note's size up or down and writes it. At either end it
// writes nothing.
func (m *Model) resize(it item, delta int) {
	current := m.sizeOf(it)
	i := 0
	for j, s := range sizes {
		if s == current {
			i = j
		}
	}
	next := sizes[max(min(i+delta, len(sizes)-1), 0)]
	if stored, _ := it.note.Doc.Get("size"); next == current && stored != "" {
		return
	}
	m.apply(it.note.Name, doc.SetKey{Key: "size", Value: next})
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
