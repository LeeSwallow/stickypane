package app

import (
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/LeeSwallow/stickypane/internal/widget"
)

// wakeMsg arrives when the time comes that a note changes by itself.
type wakeMsg struct{}

// wake asks the open notes when they next change without their file
// changing (widget.Timed: a card that stalls) and schedules one redraw for
// the earliest of those and the coming midnight, when "14:02" becomes a
// day. Nothing is scheduled again for a time already waited for, so the
// screen sleeps until something is due: time passing is one more event,
// not a timer that ticks.
func (m *Model) wake() tea.Cmd {
	now := m.now()
	y, mo, d := now.Date()
	next := time.Date(y, mo, d+1, 0, 0, 0, 0, now.Location())
	for _, it := range m.items {
		if t, ok := it.w.(widget.Timed); ok && m.isOpen(it) {
			if at := t.NextChange(now); !at.IsZero() && at.Before(next) {
				next = at
			}
		}
	}
	if next.Equal(m.wakeAt) {
		return nil
	}
	m.wakeAt = next
	return tea.Tick(next.Sub(now), func(time.Time) tea.Msg { return wakeMsg{} })
}
