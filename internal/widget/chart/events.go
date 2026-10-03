package chart

import (
	"github.com/LeeSwallow/stickypane/internal/doc"
	"github.com/LeeSwallow/stickypane/internal/widget"
)

// events says which values changed or appeared.
func events(before, after doc.Document) []widget.Event {
	was := map[string]string{}
	for _, p := range parse(before).points {
		was[p.label] = trim(p.value)
	}
	var out []widget.Event
	for _, p := range parse(after).points {
		v := trim(p.value)
		if old, had := was[p.label]; !had || old != v {
			out = append(out, widget.Event{Type: "chart.changed", Item: p.label, From: old, To: v})
		}
	}
	return out
}
