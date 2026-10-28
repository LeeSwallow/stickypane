package main

import (
	"testing"
	"time"

	"github.com/LeeSwallow/stickypane/internal/core/widget"
	"github.com/LeeSwallow/stickypane/internal/usecase/api"
)

// watch writes an event as the second it happened, the note and the type.
func TestAWatchLineIsOneFixedForm(t *testing.T) {
	at := time.Date(2026, 10, 3, 14, 2, 5, 0, time.Local)
	ev := api.Event{Time: at, Note: "work.md", Kind: "board", Event: widget.Event{Type: "card.moved", Item: "login", From: "Todo", To: "Done"}}
	if got, want := describe(ev), "14:02:05  work.md  card.moved  login  Todo → Done"; got != want {
		t.Errorf("describe = %q, want %q", got, want)
	}
}
