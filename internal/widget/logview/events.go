package logview

import (
	"strings"

	"github.com/LeeSwallow/stickypane/internal/doc"
	"github.com/LeeSwallow/stickypane/internal/widget"
)

// events says which lines were added at the end of the log.
func events(before, after doc.Document) []widget.Event {
	was := doc.SplitLines(before.Body)
	now := doc.SplitLines(after.Body)
	if len(now) <= len(was) || strings.Join(now[:len(was)], "\n") != strings.Join(was, "\n") {
		return nil // nothing added, or rewritten rather than grown
	}
	out := make([]widget.Event, 0, len(now)-len(was))
	for _, l := range now[len(was):] {
		out = append(out, widget.Event{Type: "log.appended", Item: l})
	}
	return out
}
