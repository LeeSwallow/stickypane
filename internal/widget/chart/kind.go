package chart

import (
	"github.com/LeeSwallow/stickypane/internal/doc"
	"github.com/LeeSwallow/stickypane/internal/widget"
)

// Kind registers the chart.
var Kind = widget.Kind{
	Name:     "chart",
	Label:    "Chart",
	Icon:     "▤",
	Blurb:    "Numbers as bars, a trend line, or a calendar of days.",
	Command:  "stickypane chart tokens add input 1200",
	Usage:    "Each \"label: number\" line is a value; other lines are text above the chart. \"view: spark\" draws a one-line trend and \"view: heat\" a calendar when the labels are dates. `set` replaces a value, `add` counts it up.",
	Example:  "app: 61\nboard: 31\nchecklist: 21\nstore: 16\n",
	Size:     func(doc.Document) string { return widget.SizeHalf },
	Template: func(title string) []byte { return widget.NewFile("chart", title, "") },
	Parse:    func(d doc.Document) widget.Widget { return parse(d) },
	Events:   events,
}
