package app

import (
	"path"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/LeeSwallow/stickypane/internal/store"
	"github.com/LeeSwallow/stickypane/internal/widget"
)

// newFolder is the last choice in the move dialog.
const newFolder = "New folder…"

// undo is how to take back the last delete or archive: the file went to
// from and belongs at to.
type undo struct {
	from, to string
	label    string
}

func init() {
	handlers[modeMove] = moveUpdate
	bodies[modeMove] = moveBody
	footers[modeMove] = func(m *Model) string {
		return hints(m.width, "j k", "choose", "enter", "move", "esc", "cancel")
	}

	boardKeys["u"] = func(m *Model) tea.Cmd {
		u := m.undo
		if u == nil {
			m.status = tr.NothingToUndo
			return nil
		}
		m.undo = nil
		if err := m.store.Restore(u.from, u.to); err != nil {
			m.status = say(tr.CannotRestore, map[string]any{"Name": u.label, "Where": u.from, "Err": err.Error()})
			return nil
		}
		m.reload()
		m.focusFile(u.to)
		m.status = say(tr.Restored, map[string]any{"Name": u.label})
		return nil
	}

	// m moves the file that is shown: into a folder, where it becomes a
	// page of that book, or out of its folder to the top level.
	boardKeys["m"] = onFocused(func(m *Model, it item) tea.Cmd {
		if !m.editable(it) {
			return nil
		}
		file := it.file().Name
		here := path.Dir(file)
		m.moveTargets = nil
		if here != "." {
			m.moveTargets = append(m.moveTargets, "")
		}
		for _, other := range m.items {
			if len(other.pages) > 0 && other.note.Name != here {
				m.moveTargets = append(m.moveTargets, other.note.Name)
			}
		}
		m.moveTargets = append(m.moveTargets, newFolder)
		m.moveFile, m.moveIdx = file, 0
		m.back, m.mode = m.mode, modeMove
		return nil
	})
}

func moveUpdate(m *Model, msg tea.Msg) tea.Cmd {
	k, ok := msg.(tea.KeyPressMsg)
	if !ok {
		return nil
	}
	switch k.String() {
	case "esc", "q":
		m.mode = m.back
	case "j", "down":
		m.moveIdx = min(m.moveIdx+1, len(m.moveTargets)-1)
	case "k", "up":
		m.moveIdx = max(m.moveIdx-1, 0)
	case "enter":
		target := m.moveTargets[m.moveIdx]
		m.mode = m.back
		if target != newFolder {
			m.moveTo(target)
			break
		}
		m.ask(tr.Folder, "", func(name string) {
			// A folder is named like a note: a short, plain file name.
			m.moveTo(store.Slug(name, time.Time{}))
		})
	}
	return nil
}

// moveTo moves the chosen file into folder, or to the top level when
// folder is empty, and follows it with the focus.
func (m *Model) moveTo(folder string) {
	to := path.Base(m.moveFile)
	if folder != "" {
		to = folder + "/" + to
	}
	if err := m.store.Move(m.moveFile, to); err != nil {
		m.status = say(tr.CannotMove, map[string]any{"Err": err.Error()})
		return
	}
	m.reload()
	m.focusFile(to)
}

// focusFile gives the focus to the note that holds the file called name
// and, for a book, turns to that page.
func (m *Model) focusFile(name string) {
	for i, it := range m.items {
		for p, pg := range it.pages {
			if pg.note.Name == name {
				m.anchors[it.note.Name] = anchor{page: name}
				if b, ok := it.w.(*book); ok {
					b.at = p
				}
				m.items[i].page, m.items[i].kind = p, pg.kind
				m.setFocus(it.note.Name)
				return
			}
		}
		if it.note.Name == name {
			m.setFocus(name)
			return
		}
	}
}

// moveBody draws the list of places the file can go.
func moveBody(m *Model, h int) []string {
	var body []string
	for i, t := range m.moveTargets {
		label := t
		switch t {
		case "":
			label = tr.TopLevel
		case newFolder:
			label = tr.NewFolder
		default:
			label = "▤ " + t
		}
		if i == m.moveIdx {
			body = append(body, widget.Selected.Render(widget.Pad("› "+label, moveWidth-4)))
		} else {
			body = append(body, "  "+label)
		}
	}
	base := path.Base(m.moveFile)
	title := say(tr.MoveTo, map[string]any{"Name": widget.Clean(strings.TrimSuffix(base, path.Ext(base)))})
	return dialog(title, body, moveWidth, m.width, h, m.accent())
}

// moveWidth is the move dialog at its widest.
const moveWidth = 40
