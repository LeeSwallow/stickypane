package app

import (
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/LeeSwallow/stickypane/internal/widget"
)

// helpText lists one key per line and stays under 40 cells wide, so it is
// readable in a narrow side pane. It scrolls when the pane is short.
const helpText = `stickypane keys

Board
  arrows h j k l   move focus
  tab              next note
  enter            open note
  n                jot a note
  a                add by shape
  e                edit in $EDITOR
  p                pin
  c                change color
  R                rename
  x                move to archive
  D                delete
  r                reload
  ?                this help
  q                quit

Open note
  esc              close
  e                edit in $EDITOR
  j k              move or scroll
  G                jump to the end

Open board
  h l              change column
  H L              move a card sideways
  J K              reorder a card
  n                new card

Open checklist
  space            tick an item
  n                new item`

func init() {
	handlers[modeHelp] = helpUpdate
	bodies[modeHelp] = helpBody
	footers[modeHelp] = func(*Model) string { return "j k scroll  any other key closes" }

	// Also reachable from an open note, which it returns to.
	boardKeys["?"] = func(m *Model) tea.Cmd {
		m.back, m.mode, m.helpScroll = m.mode, modeHelp, 0
		return nil
	}
}

func helpUpdate(m *Model, msg tea.Msg) tea.Cmd {
	k, ok := msg.(tea.KeyPressMsg)
	if !ok {
		return nil
	}
	if offset, scrolled := widget.ScrollKey(m.helpScroll, k.String()); scrolled {
		m.helpScroll = offset
		return nil
	}
	m.mode = m.back
	return nil
}

func helpBody(m *Model, h int) []string {
	lines := strings.Split(helpText, "\n")
	m.helpScroll = widget.ClampOffset(m.helpScroll, len(lines), h)
	return widget.Window(lines, m.helpScroll, h)
}
