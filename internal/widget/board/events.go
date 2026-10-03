package board

import (
	"github.com/LeeSwallow/stickypane/internal/doc"
	"github.com/LeeSwallow/stickypane/internal/widget"
)

// events says which cards were added, moved to another column or removed.
func events(before, after doc.Document) []widget.Event {
	was := map[string]string{}
	for _, c := range read(before).cols {
		for _, cd := range c.cards {
			if _, ok := was[cd.text]; !ok {
				was[cd.text] = c.title
			}
		}
	}
	var out []widget.Event
	seen := map[string]bool{}
	for _, c := range read(after).cols {
		for _, cd := range c.cards {
			seen[cd.text] = true
			col, had := was[cd.text]
			switch {
			case !had:
				out = append(out, widget.Event{Type: "card.added", Item: cd.text, To: c.title})
			case col != c.title:
				out = append(out, widget.Event{Type: "card.moved", Item: cd.text, From: col, To: c.title})
			}
		}
	}
	for _, c := range read(before).cols {
		for _, cd := range c.cards {
			if !seen[cd.text] {
				out = append(out, widget.Event{Type: "card.removed", Item: cd.text, From: c.title})
			}
		}
	}
	return out
}
