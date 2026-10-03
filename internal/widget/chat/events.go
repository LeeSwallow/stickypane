package chat

import (
	"github.com/LeeSwallow/stickypane/internal/doc"
	"github.com/LeeSwallow/stickypane/internal/widget"
)

// events says what was said: each message added at the end, with who.
func events(before, after doc.Document) []widget.Event {
	was, now := read(before), read(after)
	if len(now) <= len(was) {
		return nil
	}
	var out []widget.Event
	for _, m := range now[len(was):] {
		out = append(out, widget.Event{Type: "message.added", Item: m.text, From: m.who})
	}
	return out
}
