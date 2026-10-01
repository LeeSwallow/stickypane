package app

import (
	"strings"

	tea "charm.land/bubbletea/v2"
)

const helpText = `
  stickypane

  Board
    arrows, h j k l   move focus          tab   next note
    enter             open note           n     jot a note
    a                 add by shape        e     edit in $EDITOR
    p                 pin                 c     change color
    R                 rename              x     move to archive
    D                 delete              r     reload
    ?                 this help           q     quit

  Open note
    esc               close               e     edit in $EDITOR
    j k               move or scroll      G     jump to the end
    board             h l column, H L move card, J K reorder, n new card
    checklist         space toggle, n new item`

func init() {
	handlers[modeHelp] = func(m *Model, msg tea.Msg) tea.Cmd {
		if _, ok := msg.(tea.KeyPressMsg); ok {
			m.mode = modeBoard
		}
		return nil
	}
	bodies[modeHelp] = func(*Model, int) []string { return strings.Split(helpText, "\n") }
	footers[modeHelp] = func(*Model) string { return "press any key to close" }

	boardKeys["?"] = func(m *Model) tea.Cmd {
		m.mode = modeHelp
		return nil
	}
}
