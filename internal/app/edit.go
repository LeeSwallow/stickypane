package app

import (
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/LeeSwallow/stickypane/internal/editor"
	"github.com/LeeSwallow/stickypane/internal/widget"
)

func init() {
	handlers[modeEdit] = editUpdate
	bodies[modeEdit] = editBody
	footers[modeEdit] = editFooter

	// e edits the note's file here, the way vi would. It works on the
	// focused note from the main screen and from a zoomed note.
	boardKeys["e"] = onFocused(func(m *Model, it item) tea.Cmd {
		if m.zoomed() {
			if i := m.index(m.zoomName); i >= 0 {
				it = m.items[i]
			}
		}
		if !m.editable(it) {
			return nil
		}
		name := it.file().Name
		b, err := m.store.Read(name)
		if err != nil {
			m.status = say(tr.CannotReadNote, map[string]any{"Err": err.Error()})
			return nil
		}
		m.edit, m.editName, m.editDisk = editor.New(string(b)), name, string(b)
		m.editBack, m.mode = m.mode, modeEdit
		return nil
	})
}

func editUpdate(m *Model, msg tea.Msg) tea.Cmd {
	k, ok := msg.(tea.KeyPressMsg)
	if !ok {
		return nil
	}
	switch m.edit.Key(k.String(), k.Text) {
	case editor.Save:
		m.save(false)
	case editor.ForceSave:
		m.save(true)
	case editor.SaveQuit:
		if m.save(false) {
			m.leaveEditor()
		}
	case editor.Quit, editor.ForceQuit:
		m.leaveEditor()
	case editor.Reload:
		b, err := m.store.Read(m.editName)
		if err != nil {
			m.edit.SetMessage(say(tr.CannotReadFile, map[string]any{"Err": err.Error()}))
			break
		}
		m.edit.Load(string(b))
		m.editDisk = string(b)
	}
	return nil
}

// save writes the buffer to the note's file and reports whether it did. A
// file that changed since it was loaded is someone else's work, usually the
// agent's: it is overwritten only when the user insists.
func (m *Model) save(force bool) bool {
	if now, err := m.store.Read(m.editName); !force && (err != nil || string(now) != m.editDisk) {
		m.edit.SetMessage(tr.FileChangedSave)
		return false
	}
	text := m.edit.Text()
	if err := m.store.Write(m.editName, []byte(text)); err != nil {
		m.edit.SetMessage(say(tr.WriteFailed, map[string]any{"Err": err.Error()}))
		return false
	}
	m.edit.Saved()
	m.editDisk = text
	m.reload()
	return true
}

// followEdit keeps the editor honest about a file that changed on disk
// while it was open, the way resterm does: a buffer the user has not
// touched follows the file, and one with changes is kept and the user told.
func (m *Model) followEdit() {
	if m.mode != modeEdit || m.edit == nil {
		return
	}
	b, err := m.store.Read(m.editName)
	if err != nil || string(b) == m.editDisk {
		return
	}
	if m.edit.Dirty() {
		m.edit.SetMessage(tr.FileChangedEdit)
		return
	}
	m.edit.Load(string(b))
	m.editDisk = string(b)
}

func (m *Model) leaveEditor() {
	m.mode, m.edit = m.editBack, nil
	m.reload()
	if m.zoomed() && m.index(m.zoomName) < 0 {
		m.mode = modeBoard
	}
	m.reveal = revealNote
}

// editBody shows the editor in a frame that fills the screen. The border
// names the file, marks unsaved changes, and shows the cursor's place.
func editBody(m *Model, h int) []string {
	title := m.editName
	if m.edit.Dirty() {
		title += " [+]"
	}
	color := m.accent()
	if i := m.showing(m.editName); i >= 0 {
		color = m.color(m.items[i])
	}
	_, where := m.edit.Status()
	lines := m.edit.View(max(m.width-4, 1), max(h-2, 1))
	return strings.Split(frame(box{
		title: title, icon: "✎", summary: where,
		body: strings.Join(lines, "\n"), width: m.width, color: color, focused: true,
	}), "\n")
}

// editFooter is the editor's own line: the mode, the command being typed or
// a message. When it has nothing to say it shows the keys to get started.
func editFooter(m *Model) string {
	left, _ := m.edit.Status()
	switch {
	case m.edit.Mode() == editor.Command:
		return left + widget.Selected.Render(" ")
	case m.edit.Message() != "":
		return widget.Warn.Render(widget.Truncate(left, m.width))
	case left != "":
		return widget.Bold.Render(widget.Truncate(left, m.width))
	}
	return hints(m.width, "i", "insert", ":w", "save", ":q", "quit", ":wq", "save and quit", "u", "undo", "/", "search", "ctrl+f ctrl+b", "page")
}
