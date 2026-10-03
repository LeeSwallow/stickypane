package app

import (
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/LeeSwallow/stickypane/internal/doc"
	"github.com/LeeSwallow/stickypane/internal/store"
	"github.com/LeeSwallow/stickypane/internal/widget"
)

// The catalog is a dialog with the shapes on the left and, when the screen
// is wide enough, an example of the selected one on the right.
const (
	catalogWidth = 76 // the dialog at its widest
	catalogList  = 22 // the column of shape names
	catalogSplit = 60 // narrower than this, the example goes below the list
)

func init() {
	handlers[modeCatalog] = catalogUpdate
	bodies[modeCatalog] = catalogBody
	footers[modeCatalog] = func(m *Model) string {
		return hints(m.width, "j k", "choose", "enter", "pick", "esc", "cancel")
	}

	// Jot: one line becomes a plain note. No shape, no title. N always
	// jots; n jots too unless the focused open note uses n itself (a board
	// adds a card, a checklist an item).
	jot := func(m *Model) tea.Cmd {
		m.ask("Jot", "", func(text string) { m.create(text, "", []byte(text+"\n")) })
		return nil
	}
	boardKeys["N"] = jot
	boardKeys["n"] = jot
	boardKeys["a"] = func(m *Model) tea.Cmd {
		m.catalogIdx, m.mode = 0, modeCatalog
		return nil
	}
}

// create writes a new note named after text and focuses it. ext is the
// file's extension when it is not Markdown. A note made here is opened in
// sticky.json, so it is still on the screen after a restart: it is the
// user's own note, and no agent is in the middle of editing it.
func (m *Model) create(text, ext string, content []byte) {
	if ext == "" {
		ext = ".md"
	}
	name, err := m.store.Create(text, ext, content, m.now())
	if err != nil {
		m.status = "Cannot create the note: " + err.Error()
		return
	}
	open := true
	m.setView(name, func(v *store.View) { v.Open = &open })
	m.setFocus(name)
}

func catalogUpdate(m *Model, msg tea.Msg) tea.Cmd {
	k, ok := msg.(tea.KeyPressMsg)
	if !ok {
		return nil
	}
	switch k.String() {
	case "esc", "q":
		m.mode = modeBoard
	case "j", "down":
		m.catalogIdx = min(m.catalogIdx+1, len(m.reg)-1)
	case "k", "up":
		m.catalogIdx = max(m.catalogIdx-1, 0)
	case "enter":
		kind := m.reg[m.catalogIdx]
		m.mode = modeBoard
		m.ask("Title", "", func(title string) { m.create(title, kind.New, kind.Template(title)) })
	}
	return nil
}

// catalogBody draws the shape picker: every shape by name, what the selected
// one is for, and a small example of it drawn by its own widget.
func catalogBody(m *Model, h int) []string {
	kind := m.reg[m.catalogIdx]
	var list []string
	for i, k := range m.reg {
		name := strings.TrimSpace(k.Icon + " " + k.Label)
		if i == m.catalogIdx {
			list = append(list, widget.Selected.Render(widget.Pad("› "+name, catalogList-1)))
		} else {
			list = append(list, "  "+name)
		}
	}

	boxW := min(catalogWidth, m.width)
	inner := boxW - 4
	side := inner - catalogList
	if m.width < catalogSplit {
		side = inner
	}
	var detail []string
	if side >= 16 {
		for _, l := range widget.Wrap(kind.Blurb, side) {
			detail = append(detail, widget.Faint.Render(l))
		}
		if kind.Example != "" && kind.Parse != nil {
			example, _ := kind.Parse(doc.Parse([]byte(kind.Example))).Draw(side-4, false)
			detail = append(detail, "")
			detail = append(detail, strings.Split(frame(box{
				title: "example", icon: kind.Icon, body: example, width: side, color: m.noteColor(m.catalogIdx),
			}), "\n")...)
		}
	}

	var body []string
	if m.width < catalogSplit {
		body = append(append(list, ""), detail...)
	} else {
		for r := 0; r < max(len(list), len(detail)); r++ {
			left, right := "", ""
			if r < len(list) {
				left = list[r]
			}
			if r < len(detail) {
				right = detail[r]
			}
			body = append(body, widget.Pad(left, catalogList)+right)
		}
	}
	return dialog("Add a note", body, boxW, m.width, h, m.accent())
}
