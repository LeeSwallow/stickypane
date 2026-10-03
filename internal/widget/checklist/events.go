package checklist

import (
	"github.com/LeeSwallow/stickypane/internal/doc"
	"github.com/LeeSwallow/stickypane/internal/widget"
)

// events says which items were ticked, unticked, added or removed.
func events(before, after doc.Document) []widget.Event {
	was := map[string]bool{}
	b := read(before)
	for _, i := range b.items {
		was[b.entries[i].text] = b.entries[i].checked
	}
	var out []widget.Event
	a := read(after)
	seen := map[string]bool{}
	for _, i := range a.items {
		e := a.entries[i]
		seen[e.text] = true
		checked, had := was[e.text]
		switch {
		case !had:
			out = append(out, widget.Event{Type: "item.added", Item: e.text})
		case !checked && e.checked:
			out = append(out, widget.Event{Type: "item.ticked", Item: e.text})
		case checked && !e.checked:
			out = append(out, widget.Event{Type: "item.unticked", Item: e.text})
		}
	}
	for _, i := range b.items {
		if t := b.entries[i].text; !seen[t] {
			out = append(out, widget.Event{Type: "item.removed", Item: t})
		}
	}
	return out
}
