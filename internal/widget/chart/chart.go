// Package chart is the numbers note: "label: number" lines drawn as bars, as
// a one-line trend, or as a calendar of days.
package chart

import (
	"fmt"
	"math"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/LeeSwallow/stickypane/internal/doc"
	"github.com/LeeSwallow/stickypane/internal/widget"
)

const (
	formatHint = `no data yet: add "label: number" lines`
	dayLayout  = "2006-01-02"
	gutterDays = 5 // cells for the weekday names left of a calendar
)

// dataRe matches a data line: an optional list marker, a label, a colon,
// a space and a number that may use "," or "_" between digits. The space is
// required, so that a time ("12:30") or an address ("host:8080") in a line
// of text is not taken for data.
var dataRe = regexp.MustCompile(`^\s*(?:[-*]\s+)?(.+?)\s*:\s+(-?[0-9][0-9,_]*(?:\.[0-9]+)?)\s*$`)

// Kind registers the chart.
var Kind = widget.Kind{
	Name:     "chart",
	Label:    "Chart",
	Icon:     "▤",
	Blurb:    "Numbers as bars, a trend line, or a calendar of days.",
	Example:  "app: 61\nboard: 31\nchecklist: 21\nstore: 16\n",
	Size:     func(doc.Document) string { return widget.SizeHalf },
	Template: func(title string) []byte { return widget.NewFile("chart", title, "") },
	Parse:    func(d doc.Document) widget.Widget { return parse(d) },
}

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

// Draw implements widget.Widget. A chart has no cursor.
func (c *Chart) Draw(width int, _ bool) (string, widget.Span) {
	var lines []string
	if len(c.points) == 0 {
		lines = append(lines, widget.Faint.Render(widget.Truncate(formatHint, width)))
	}
	for _, p := range c.prose {
		lines = append(lines, widget.Wrap(widget.Clean(p), max(width, 1))...)
	}
	if len(c.points) > 0 {
		if len(c.prose) > 0 {
			lines = append(lines, "")
		}
		switch days := c.days(); {
		case c.view == "spark":
			lines = append(lines, c.spark(width)...)
		case c.view == "heat" && len(days) > 0:
			lines = append(lines, heat(days, width)...)
		default:
			lines = append(lines, c.bars(width)...)
		}
	}
	return widget.Fit(strings.Join(lines, "\n"), width), widget.NoSpan
}

// eighths are the partial blocks that end a bar, from one eighth of a cell.
var eighths = []string{"", "▏", "▎", "▍", "▌", "▋", "▊", "▉"}

// bars draws one row per value: the label, a bar scaled to the largest
// value, and the number as it was written.
func (c *Chart) bars(width int) []string {
	labelW, valueW, top := 0, 0, 0.0
	for _, p := range c.points {
		labelW = max(labelW, widget.Width(widget.Clean(p.label)))
		valueW = max(valueW, widget.Width(p.text))
		top = math.Max(top, p.value)
	}
	labelW = min(labelW, max(width/3, 4))
	barW := width - labelW - valueW - 2
	lines := make([]string, 0, len(c.points))
	for _, p := range c.points {
		label := widget.Pad(widget.Clean(p.label), labelW)
		value := strings.Repeat(" ", valueW-widget.Width(p.text)) + p.text
		if barW < 3 {
			lines = append(lines, label+" "+value) // no room for bars
			continue
		}
		units := 0
		if top > 0 && p.value > 0 {
			units = int(math.Round(p.value / top * float64(barW*8)))
		}
		bar := strings.Repeat("█", units/8) + eighths[units%8]
		lines = append(lines, label+" "+widget.Good.Render(bar)+strings.Repeat(" ", barW-widget.Width(bar))+" "+value)
	}
	return lines
}

// sparks are the eight heights of a trend line.
var sparks = []rune("▁▂▃▄▅▆▇█")

// spark draws the values as one line, the latest that fit, and names the
// range and the peak under it.
func (c *Chart) spark(width int) []string {
	points := c.points
	if width > 0 && len(points) > width {
		points = points[len(points)-width:]
	}
	low, high := points[0], points[0]
	for _, p := range points {
		if p.value < low.value {
			low = p
		}
		if p.value > high.value {
			high = p
		}
	}
	var line strings.Builder
	for _, p := range points {
		i := len(sparks) / 2
		// A range too large for a float gives a ratio that is not a
		// number; such a value sits in the middle.
		if ratio := (p.value - low.value) / (high.value - low.value); high.value > low.value && !math.IsNaN(ratio) && !math.IsInf(ratio, 0) {
			i = int(math.Round(ratio * float64(len(sparks)-1)))
		}
		line.WriteRune(sparks[max(min(i, len(sparks)-1), 0)])
	}
	legend := fmt.Sprintf("%s → %s  low %s  peak %s (%s)",
		widget.Clean(points[0].label), widget.Clean(points[len(points)-1].label), low.text, high.text, widget.Clean(high.label))
	return append([]string{widget.Good.Render(line.String())}, faintLines(legend, width)...)
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

// shades are the four levels of a calendar cell, from light to full.
var shades = []string{"░", "▒", "▓", "█"}

// heat draws a calendar like a contribution graph: one column per week, one
// row per weekday, darker for larger values. It shows the latest weeks that
// fit. A day with a zero is a dot; a day without a value is empty.
//
// A day's shade is its rank among the days with a value, not its share of
// the peak: one day far above the others would otherwise leave every other
// day at the lightest shade.
func heat(days []day, width int) []string {
	values := map[string]point{}
	var ranked []float64
	first, last, peak := days[0].date, days[0].date, days[0]
	total := 0.0
	for _, d := range days {
		values[d.date.Format(dayLayout)] = d.p
		if d.date.Before(first) {
			first = d.date
		}
		if d.date.After(last) {
			last = d.date
		}
		if d.p.value > peak.p.value {
			peak = d
		}
		total += d.p.value
		if d.p.value > 0 {
			ranked = append(ranked, d.p.value)
		}
	}
	sort.Float64s(ranked)
	monday := func(t time.Time) time.Time { return t.AddDate(0, 0, -((int(t.Weekday()) + 6) % 7)) }
	start, end := monday(first), monday(last)
	weeks := int(end.Sub(start).Hours()/24/7) + 1
	if fit := max((width-gutterDays)/2, 1); weeks > fit {
		start = end.AddDate(0, 0, -7*(fit-1))
		weeks = fit
	}

	head := strings.Repeat(" ", gutterDays)
	month := time.Month(0)
	for w := 0; w < weeks; w++ {
		cell := "  "
		if m := start.AddDate(0, 0, 7*w).Month(); m != month {
			cell = widget.Pad(strconv.Itoa(int(m)), 2)
			month = m
		}
		head += cell
	}
	lines := []string{widget.Faint.Render(strings.TrimRight(head, " "))}
	names := []string{"Mon", "", "Wed", "", "Fri", "", ""}
	for row := 0; row < 7; row++ {
		var cells strings.Builder
		for w := 0; w < weeks; w++ {
			p, ok := values[start.AddDate(0, 0, 7*w+row).Format(dayLayout)]
			switch {
			case !ok:
				cells.WriteString("  ")
			case p.value <= 0:
				cells.WriteString(widget.Faint.Render("·") + " ")
			default:
				atOrBelow := sort.SearchFloat64s(ranked, math.Nextafter(p.value, math.Inf(1)))
				level := (atOrBelow*len(shades)+len(ranked)-1)/len(ranked) - 1
				cells.WriteString(widget.Good.Render(shades[max(min(level, len(shades)-1), 0)]) + " ")
			}
		}
		line := widget.Faint.Render(widget.Pad(names[row], gutterDays)) + cells.String()
		lines = append(lines, strings.TrimRight(line, " "))
	}
	legend := fmt.Sprintf("total %s · peak %s (%s)", trim(total), peak.p.text, peak.date.Format(dayLayout))
	return append(append(lines, ""), faintLines(legend, width)...)
}

// faintLines wraps a legend to width and dims it.
func faintLines(text string, width int) []string {
	lines := widget.Wrap(text, max(width, 1))
	for i, l := range lines {
		lines[i] = widget.Faint.Render(l)
	}
	return lines
}

// trim formats a sum without a needless ".0" and groups its thousands.
func trim(v float64) string {
	whole, frac, _ := strings.Cut(strconv.FormatFloat(math.Round(v*100)/100, 'f', -1, 64), ".")
	sign := ""
	if strings.HasPrefix(whole, "-") {
		sign, whole = "-", whole[1:]
	}
	for i := len(whole) - 3; i > 0; i -= 3 {
		whole = whole[:i] + "," + whole[i:]
	}
	if frac != "" {
		frac = "." + frac
	}
	return sign + whole + frac
}

// Summary implements widget.Widget: how many values the chart has.
func (c *Chart) Summary() string {
	switch n := len(c.points); n {
	case 0:
		return ""
	case 1:
		return "1 value"
	default:
		return fmt.Sprintf("%d values", n)
	}
}

// Update implements widget.Widget. A chart takes no keys.
func (c *Chart) Update(string) (widget.Widget, widget.Result) { return c, widget.Result{} }

// Sync implements widget.Widget.
func (c *Chart) Sync(d doc.Document) widget.Widget { return parse(d) }
