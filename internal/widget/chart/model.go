package chart

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/LeeSwallow/stickypane/internal/doc"
	"github.com/LeeSwallow/stickypane/internal/widget"
)

type point struct {
	label string
	text  string // the number as written
	value float64
}

// Chart is the widget for a chart note.
type Chart struct {
	view   string // "bar", "spark" or "heat"
	points []point
	prose  []string // the lines that are not data, shown above the chart
}

func parse(d doc.Document) *Chart {
	c := &Chart{}
	c.view, _ = d.Get("view")
	c.view = strings.ToLower(c.view)
	for _, line := range doc.Lines(d.Body) {
		line = strings.TrimRight(line, "\r")
		if m := dataRe.FindStringSubmatch(line); m != nil {
			if v, err := strconv.ParseFloat(strings.NewReplacer(",", "", "_", "").Replace(m[2]), 64); err == nil {
				c.points = append(c.points, point{label: m[1], text: m[2], value: v})
				continue
			}
		}
		c.prose = append(c.prose, strings.TrimRight(line, " \t"))
	}
	if days := c.days(); c.view == "heat" && len(days) > 0 && len(days) < len(c.points) {
		// A calendar draws days only. A value that is not a day is shown
		// as the text it is rather than dropped.
		var dated []point
		for _, p := range c.points {
			if _, err := time.Parse(dayLayout, strings.TrimSpace(p.label)); err == nil {
				dated = append(dated, p)
			} else {
				c.prose = append(c.prose, p.label+": "+p.text)
			}
		}
		c.points = dated
	}
	for len(c.prose) > 0 && c.prose[len(c.prose)-1] == "" {
		c.prose = c.prose[:len(c.prose)-1]
	}
	for len(c.prose) > 0 && c.prose[0] == "" {
		c.prose = c.prose[1:]
	}
	return c
}

// day is one dated value.
type day struct {
	date time.Time
	p    point
}

// days returns the points whose label is a date such as 2026-10-01, in the
// order they were written.
func (c *Chart) days() []day {
	var days []day
	for _, p := range c.points {
		if t, err := time.Parse(dayLayout, strings.TrimSpace(p.label)); err == nil {
			days = append(days, day{t, p})
		}
	}
	return days
}

// Summary implements widget.Widget: how many values the chart has.
func (c *Chart) Summary() string {
	switch n := len(c.points); n {
	case 0:
		return ""
	case 1:
		return widget.T("1 value")
	default:
		return fmt.Sprintf(widget.T("%d values"), n)
	}
}

// Value returns the value called label as the chart's file writes it.
func Value(d doc.Document, label string) (string, bool) {
	lines := doc.Lines(d.Body)
	if i, from, to := find(lines, strings.TrimSpace(label)); i >= 0 {
		return lines[i][from:to], true
	}
	return "", false
}
