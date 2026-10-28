package chart

import (
	"fmt"
	"strings"

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
	Options: []widget.Option{
		{Name: "view", Type: widget.OptChoice, Label: "View", Choices: []string{"bar", "spark", "heat"}, Default: "bar"},
		{Name: "values", Type: widget.OptLines, Label: "Values"},
		{Name: "from", Type: widget.OptText, Label: "Count from note"},
	},
	Compose: compose,
	Parse:   func(d doc.Document) widget.Widget { return parse(d) },
	Events:  events,
	Data:    data,
}

// compose writes a new chart: its view when it is not bars, the note it
// counts from when it has one, and each "label: number" value.
func compose(title string, opts map[string]string) ([]byte, error) {
	var b strings.Builder
	for _, l := range widget.Lines(opts["values"]) {
		m := dataRe.FindStringSubmatch(l)
		if m == nil {
			return nil, fmt.Errorf("%w: values of a chart: %q is not \"label: number\"", widget.ErrBadOption, l)
		}
		b.WriteString(m[1] + ": " + m[2] + "\n")
	}
	d := doc.Parse(widget.NewFile("chart", title, b.String()))
	if v := opts["view"]; v != "" && v != "bar" {
		d = d.Set("view", v)
	}
	if from := opts["from"]; from != "" {
		d = d.Set("from", from)
	}
	return d.Bytes(), nil
}
