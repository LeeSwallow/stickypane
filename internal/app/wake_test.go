package app

import (
	"strings"
	"testing"
	"time"
)

// Time passing is an event too: the board asks its notes when they next
// change by themselves (a card that stalls) and wakes then, once, not on a
// timer; and at midnight, when "14:02" becomes a day.
func TestTheBoardWakesWhenACardWillStall(t *testing.T) {
	clock := time.Date(2026, 10, 3, 15, 0, 0, 0, time.Local)
	board := "---\ntype: board\nopen: true\n---\n## Doing\n- slow @codex-1 @{2026-10-03} @@{14:50}\n## Done\n"
	m, dir := newModel(t, map[string]string{"b.md": board})
	m.now = func() time.Time { return clock }
	m.drawn = nil // drawn at the real time when the model was made
	m.Update(changedMsg{})
	if want := time.Date(2026, 10, 3, 15, 20, 0, 0, time.Local); !m.wakeAt.Equal(want) {
		t.Fatalf("the board should wake at 15:20, when the card stalls; wakes at %v", m.wakeAt)
	}
	if strings.Contains(screen(m), "⚠") {
		t.Fatalf("not stalled yet:\n%s\n%s", screen(m), readFile(t, dir, "b.md"))
	}
	clock = clock.Add(25 * time.Minute)
	stallClock(m, clock)
	m.Update(wakeMsg{})
	if !strings.Contains(screen(m), "⚠") {
		t.Errorf("at the wake the card shows it stalled:\n%s", screen(m))
	}
	if want := time.Date(2026, 10, 4, 0, 0, 0, 0, time.Local); !m.wakeAt.Equal(want) {
		t.Errorf("then nothing waits but midnight; wakes at %v", m.wakeAt)
	}
}

// stallClock moves the clock the board's widgets read.
func stallClock(m *Model, t time.Time) {
	m.now = func() time.Time { return t }
}
