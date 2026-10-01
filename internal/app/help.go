package app

import (
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/LeeSwallow/stickypane/internal/widget"
)

// helpText lists one key per line and stays under 40 cells wide, so it is
// readable in a narrow side pane. It scrolls when the pane is short.
const helpText = `Notes
  tab              next note
  shift+tab        previous note
  enter            open, then zoom
  z                zoom
  o                open or close
  + -              bigger, smaller
  { }              move earlier, later
  , .              turn a folder's pages
  m                move to a folder
  N                jot a note
  a                add by shape
  e                edit here (like vi)
  E                edit in $EDITOR
  p                pin
  c                change color
  R                rename
  x                move to archive
  D                delete (to .trash)
  u                undo x or D
  r                reload
  j k g G          scroll in the note
  pgdn pgup        a page down, up
  [ ]              previous, next screen
  ?                this help
  q                quit

Zoomed note
  esc              back
  j k g G          scroll

Editing a note
  i a o            insert text
  esc              back to commands
  h j k l w b      move
  x dd D u         delete, undo
  /text  n         search
  ctrl+f ctrl+b    next, previous page
  :w  :q  :wq      save, quit, both

Open board
  h l              change column
  j k              change card
  H L              move a card sideways
  J K              reorder a card
  n                new card

Open checklist
  j k              change item
  space            tick an item
  n                new item

Open form
  j k              change control
  enter space      choose, type, press

Mouse
  click a title    open, focus, close
  click a note     focus; tick, choose
  double click     zoom
  wheel            scroll that note`

// helpWidth is the dialog that holds helpText: its widest line plus the frame.
const helpWidth = 44

func init() {
	handlers[modeHelp] = helpUpdate
	bodies[modeHelp] = helpBody
	footers[modeHelp] = func(m *Model) string {
		return hints(m.width, "j k", "scroll") + "  " + widget.Faint.Render("any other key closes")
	}

	// Also reachable from a zoomed note, which it returns to.
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
	if offset, scrolled := widget.ScrollKey(m.helpScroll, k.String(), max(m.height-4, 1)); scrolled {
		m.helpScroll = offset
		return nil
	}
	m.mode = m.back
	return nil
}

// helpBody draws the key reference: section names stand out and the keys
// line up. On a screen with room it sits in a centered dialog.
func helpBody(m *Model, h int) []string {
	lines := strings.Split(helpText, "\n")
	for i, l := range lines {
		if l != "" && !strings.HasPrefix(l, " ") {
			lines[i] = widget.Bold.Render(l)
		}
	}
	rows := h
	framed := m.width >= helpWidth && h >= 6
	if framed {
		rows = min(h-2, len(lines))
	}
	m.helpScroll = widget.ClampOffset(m.helpScroll, len(lines), rows)
	visible := append([]string(nil), widget.Window(lines, m.helpScroll, rows)...)
	if !framed {
		return visible
	}
	return dialog("Keys", visible, helpWidth, m.width, h)
}
