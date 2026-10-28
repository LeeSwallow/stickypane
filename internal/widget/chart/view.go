// Package chart is the numbers note: "label: number" lines drawn as bars, as
// a one-line trend, or as a calendar of days.
package chart

import (
	"fmt"
	"math"
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

	sparkRows    = 6  // height of a trend line when nothing says more
	maxSparkRows = 40 // and the most a zoomed one grows to
	sparkMinW    = 24 // narrower than this a trend is drawn on one line
	sparkMinPlot = 12 // cells the plot needs, or the trend is one line
	heatMinW     = 40 // narrower than this a calendar keeps two cells a week
)

// Draw implements widget.Widget. A chart has no cursor.
func (c *Chart) Draw(width int, active bool) (string, widget.Span) {
	return c.DrawIn(width, 0, active)
}

// DrawIn implements widget.Roomy: the same drawing, with a trend line as
// tall as height allows. A pane does not know its height before it is
// drawn, so Draw passes 0 and the trend line has its default height; the
// zoomed screen passes the rows it has.
func (c *Chart) DrawIn(width, height int, _ bool) (string, widget.Span) {
	var lines []string
	if len(c.points) == 0 {
		lines = append(lines, widget.Faint.Render(widget.Truncate(widget.T(formatHint), width)))
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
			rows := sparkRows
			if height > 0 {
				// The rows the room has left under the text, less the
				// line of labels under the chart.
				rows = max(min(height-len(lines)-1, maxSparkRows), sparkRows)
			}
			lines = append(lines, c.spark(width, rows)...)
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

// ratio is where v lies between low and high, 0 to 1. A range too large
// for a float gives a ratio that is not a number; such a value sits in the
// middle.
func ratio(v, low, high float64) float64 {
	if r := (v - low) / (high - low); high > low && !math.IsNaN(r) && !math.IsInf(r, 0) {
		return r
	}
	return 0.5
}

// spark draws the values as a line chart rows high that fills the width:
// the largest and smallest on an axis at the left, the first and last
// labels under it, and the latest value marked and written at the end.
// With fewer than two values, or too little width, it is one line.
func (c *Chart) spark(width, rows int) []string {
	last := c.points[len(c.points)-1]
	valueW := min(widget.Width(last.text), max(width/4, 1))
	axisW := 0
	for _, p := range c.points {
		axisW = max(axisW, min(widget.Width(p.text), max(width/5, 1)))
	}
	plotW := width - axisW - 1 - valueW - 1
	if len(c.points) < 2 || width < sparkMinW || plotW < sparkMinPlot || rows < 2 {
		return c.oneLine(width)
	}
	points := c.points
	if len(points) > plotW {
		points = points[len(points)-plotW:]
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
	ratios := make([]float64, len(points))
	for i, p := range points {
		ratios[i] = ratio(p.value, low.value, high.value)
	}
	// Each column is the line's height there, found between the two
	// values on either side, so a few values stretch over the width.
	levels := make([]int, plotW)
	for x := range levels {
		pos := float64(x) * float64(len(points)-1) / float64(plotW-1)
		i := min(int(pos), len(points)-2)
		f := pos - float64(i)
		levels[x] = int(math.Round((ratios[i]*(1-f)+ratios[i+1]*f)*float64(rows*8-1))) + 1
	}
	top := (levels[plotW-1] - 1) / 8 // the row the latest value is on

	lines := make([]string, 0, rows+1)
	for row := rows - 1; row >= 0; row-- {
		var cells strings.Builder
		for x, level := range levels {
			k := level - row*8
			switch {
			case x == plotW-1 && row == top:
				cells.WriteString(widget.Accent.Render("●"))
			case k >= 8:
				cells.WriteString(widget.Good.Render("█"))
			case k > 0:
				cells.WriteString(widget.Good.Render(string(sparks[k-1])))
			default:
				cells.WriteString(" ")
			}
		}
		label, mark := "", "│"
		switch row {
		case rows - 1:
			label, mark = high.text, "┤"
		case 0:
			label, mark = low.text, "┤"
		}
		tail := ""
		if row == top {
			tail = " " + widget.Accent.Render(widget.Truncate(last.text, valueW))
		}
		label = strings.Repeat(" ", axisW-min(widget.Width(label), axisW)) + widget.Truncate(label, axisW)
		lines = append(lines, widget.Faint.Render(label+mark)+cells.String()+tail)
	}
	first, end := widget.Clean(points[0].label), widget.Clean(last.label)
	if widget.Width(first)+widget.Width(end)+1 > plotW {
		first, end = widget.Truncate(first, (plotW-1)/2), widget.Truncate(end, (plotW-1)/2)
	}
	gap := max(plotW-widget.Width(first)-widget.Width(end), 1)
	return append(lines, widget.Faint.Render(strings.Repeat(" ", axisW+1)+first+strings.Repeat(" ", gap)+end))
}

// oneLine draws the values as one line, the latest that fit, and names the
// range and the peak under it.
func (c *Chart) oneLine(width int) []string {
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
		i := int(math.Round(ratio(p.value, low.value, high.value) * float64(len(sparks)-1)))
		line.WriteRune(sparks[max(min(i, len(sparks)-1), 0)])
	}
	legend := fmt.Sprintf("%s → %s  low %s  peak %s (%s)",
		widget.Clean(points[0].label), widget.Clean(points[len(points)-1].label), low.text, high.text, widget.Clean(high.label))
	return append([]string{widget.Good.Render(line.String())}, faintLines(legend, width)...)
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
	// A week is 2 to 4 cells wide: the widest that shows every week, or 2
	// with the latest weeks that fit. Narrow, it is always 2.
	cw := 2
	if width >= heatMinW {
		for cw < 4 && weeks*(cw+1) <= width-gutterDays {
			cw++
		}
	}
	if fit := max((width-gutterDays)/cw, 1); weeks > fit {
		start = end.AddDate(0, 0, -7*(fit-1))
		weeks = fit
	}

	head := strings.Repeat(" ", gutterDays)
	month := time.Month(0)
	for w := 0; w < weeks; w++ {
		cell := strings.Repeat(" ", cw)
		if m := start.AddDate(0, 0, 7*w).Month(); m != month {
			cell = widget.Pad(strconv.Itoa(int(m)), cw)
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
				cells.WriteString(strings.Repeat(" ", cw))
			case p.value <= 0:
				cells.WriteString(widget.Faint.Render("·") + strings.Repeat(" ", cw-1))
			default:
				atOrBelow := sort.SearchFloat64s(ranked, math.Nextafter(p.value, math.Inf(1)))
				level := (atOrBelow*len(shades)+len(ranked)-1)/len(ranked) - 1
				// Two cells a week leave a gap; on a wide pane weeks are
				// blocks that touch, so the shades can be told apart.
				block := shades[max(min(level, len(shades)-1), 0)]
				if width >= heatMinW {
					cells.WriteString(widget.Good.Render(strings.Repeat(block, cw)))
				} else {
					cells.WriteString(widget.Good.Render(block) + " ")
				}
			}
		}
		line := widget.Faint.Render(widget.Pad(names[row], gutterDays)) + cells.String()
		lines = append(lines, strings.TrimRight(line, " "))
	}
	legend := fmt.Sprintf("total %s · peak %s (%s)", trim(total), peak.p.text, peak.date.Format(dayLayout))
	lines = append(lines, "")
	if width < heatMinW {
		return append(lines, faintLines(legend, width)...)
	}
	// The scale sits next to the legend when both fit on a line.
	scale := widget.T("less") + " " + strings.Join(shades, "") + " " + widget.T("more")
	if widget.Width(legend)+2+widget.Width(scale) > width {
		return append(append(lines, faintLines(legend, width)...), widget.Faint.Render(widget.Truncate(scale, width)))
	}
	return append(lines, widget.Faint.Render(legend+"  "+scale))
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

// Update implements widget.Widget. A chart takes no keys.
func (c *Chart) Update(string) (widget.Widget, widget.Result) { return c, widget.Result{} }

// Sync implements widget.Widget.
func (c *Chart) Sync(d doc.Document) widget.Widget { return parse(d) }
