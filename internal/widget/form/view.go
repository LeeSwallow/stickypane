// Package form is the document that asks: Markdown with options to choose,
// lines to fill in and buttons to press. The answers are written into the
// same file, so the agent that wrote the questions reads them back.
package form

import (
	"strings"
	"time"

	"github.com/LeeSwallow/stickypane/internal/when"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/LeeSwallow/stickypane/internal/doc"
	"github.com/LeeSwallow/stickypane/internal/widget"
	"github.com/LeeSwallow/stickypane/internal/widget/note"
)

const (
	// defaultButton is the button of a form that has none of its own.
	defaultButton = "Submit"
	fieldWidth    = 48 // widest a text field gets, borders included
	placeholder   = "enter to type"

	keySubmitted   = "submitted"
	keySubmittedAt = "submitted_at"
)

// now is the clock, replaced in tests.
var now = func() time.Time { return widget.Now() }

// Form is the widget for a form note: the content, how its prose is drawn
// and where the cursor is.
type Form struct {
	content
	render note.Renderer
	cursor int
	drawn  []area // where each control was in the last Draw
}

// area is the cells a control took in the last Draw: its lines and, for a
// button, its columns.
type area struct {
	lines  widget.Span
	x0, x1 int
}

// anyCol is the column range of a control that owns its whole lines.
const anyCol = 1 << 30

func (f *Form) clamp() { f.cursor = max(min(f.cursor, len(f.controls)-1), 0) }

// nth is how many controls of the same kind and text come before control c.
func (f *Form) nth(c control) int {
	n := 0
	for i := 0; i < c.line; i++ {
		l, target := f.lines[i], f.lines[c.line]
		if l.kind == target.kind && (target.kind == field || l.text == target.text) {
			n++
		}
	}
	return n
}

// Draw implements widget.Widget. Text is drawn like a plain note; options,
// fields and buttons are drawn as controls where they are written.
func (f *Form) Draw(width int, active bool) (string, widget.Span) {
	f.clamp()
	var out []string
	at := widget.NoSpan
	f.drawn = f.drawn[:0]
	sel := f.controls[f.cursor]
	for i := 0; i < len(f.lines); i++ {
		l := f.lines[i]
		selected := active && sel.line == i
		from := len(out)
		switch l.kind {
		case prose:
			end := i
			for end < len(f.lines) && f.lines[end].kind == prose {
				end++
			}
			out = append(out, f.text(f.lines[i:end], width, i > 0, end < len(f.lines))...)
			i = end - 1
		case option:
			out = append(out, drawOption(l, width, selected)...)
			for i+1 < len(f.lines) && f.lines[i+1].kind == detail {
				i++
				for _, d := range widget.Wrap(widget.Clean(f.lines[i].text), max(width-4, 1)) {
					out = append(out, "    "+widget.Faint.Render(d))
				}
			}
		case detail: // a detail with no option above it cannot happen; show it
			out = append(out, widget.Wrap(widget.Clean(l.text), max(width, 1))...)
		case field:
			out = append(out, drawField(l.text, width, selected)...)
		case buttons:
			focus := -1
			if selected {
				focus = sel.button
			}
			rows, places := f.drawButtons(l.labels, width, focus)
			starts := make([]int, len(rows))
			for r, row := range rows {
				starts[r] = len(out)
				out = append(out, row...)
			}
			for b, p := range places {
				lines := widget.Span{Start: starts[p.row], End: starts[p.row] + len(rows[p.row])}
				f.drawn = append(f.drawn, area{lines: lines, x0: p.x0, x1: p.x1})
				if b == focus {
					at = lines
				}
			}
			continue
		}
		if l.kind == option || l.kind == field {
			lines := widget.Span{Start: from, End: len(out)}
			f.drawn = append(f.drawn, area{lines: lines, x1: anyCol})
			if selected {
				at = lines
			}
		}
	}
	if f.submitted != "" {
		out = append(out, "", f.sentRule(width))
		// What the form produced: the answers, as stickypane answers says them.
		for _, a := range collect(f.lines, f.questions) {
			value := "-"
			if len(a.Values) > 0 {
				value = strings.Join(a.Values, "; ")
			}
			text := widget.Clean(value)
			if a.Question != "" {
				text = widget.Clean(a.Question) + ": " + text
			}
			out = append(out, widget.Wrap(text, max(width, 1))...)
		}
	}
	return widget.Fit(strings.Join(out, "\n"), width), at
}

// text draws a run of prose lines. The blank lines at its ends are the
// form's to decide, not the renderer's: they become one blank line each, and
// only where something is drawn on both sides. before and after report
// whether anything comes before and after the run.
func (f *Form) text(lines []line, width int, before, after bool) []string {
	first, last := 0, len(lines)
	for first < last && strings.TrimSpace(lines[first].text) == "" {
		first++
	}
	for last > first && strings.TrimSpace(lines[last-1].text) == "" {
		last--
	}
	if first == last {
		if before && after {
			return []string{""}
		}
		return nil
	}
	raw := make([]string, 0, last-first)
	for _, l := range lines[first:last] {
		raw = append(raw, widget.Clean(l.text))
	}
	drawn := strings.Split(f.render(strings.Join(raw, "\n")+"\n", width), "\n")
	blank := func(s string) bool { return strings.TrimSpace(ansi.Strip(s)) == "" }
	for len(drawn) > 0 && blank(drawn[0]) {
		drawn = drawn[1:]
	}
	for len(drawn) > 0 && blank(drawn[len(drawn)-1]) {
		drawn = drawn[:len(drawn)-1]
	}
	var out []string
	if before && first > 0 {
		out = append(out, "")
	}
	out = append(out, drawn...)
	if after && last < len(lines) {
		out = append(out, "")
	}
	return out
}

func drawOption(l line, width int, selected bool) []string {
	glyph := "☐ "
	switch {
	case l.radio && l.on:
		glyph = "◉ "
	case l.radio:
		glyph = "○ "
	case l.on:
		glyph = "☑ "
	}
	var out []string
	for j, text := range widget.Wrap(widget.Clean(l.text), max(width-4, 1)) {
		lead := "    "
		if j == 0 {
			lead = "  " + glyph
			if selected {
				lead = "› " + glyph
			}
		}
		switch {
		case selected:
			out = append(out, widget.Selected.Render(lead+text))
		case l.on:
			out = append(out, widget.Good.Render(lead)+widget.Bold.Render(text))
		default:
			out = append(out, lead+text)
		}
	}
	return out
}

// border is the six pieces of a box: corners, the horizontal and the
// vertical line.
type border struct{ tl, tr, bl, br, h, v string }

var (
	thin  = border{"╭", "╮", "╰", "╯", "─", "│"}
	heavy = border{"┏", "┓", "┗", "┛", "━", "┃"}
)

// boxed draws lines inside a box whose inside is inner cells wide. The
// border and the text are styled separately so a focused box can stand out.
func boxed(lines []string, inner int, b border, edge, text lipgloss.Style) []string {
	rule := strings.Repeat(b.h, inner+2)
	out := []string{edge.Render(b.tl + rule + b.tr)}
	for _, l := range lines {
		out = append(out, edge.Render(b.v)+text.Render(" "+widget.Pad(l, inner)+" ")+edge.Render(b.v))
	}
	return append(out, edge.Render(b.bl+rule+b.br))
}

// drawField draws a line to fill in as a box with the answer inside.
func drawField(text string, width int, selected bool) []string {
	inner := min(width, fieldWidth) - 4
	if inner < 1 {
		return []string{widget.Truncate("✎ "+widget.Clean(text), width)}
	}
	lines, st := widget.Wrap(widget.Clean(text), inner), lipgloss.NewStyle()
	if text == "" {
		lines, st = []string{""}, widget.Faint
		if selected {
			lines = []string{widget.Truncate(widget.T(placeholder), inner)}
		}
	}
	if selected {
		return boxed(lines, inner, heavy, widget.Bold, st)
	}
	return boxed(lines, inner, thin, widget.Faint, st)
}

// place is where a button was drawn: its row and its columns.
type place struct{ row, x0, x1 int }

// drawButtons draws the buttons of one line side by side, on more rows when
// they do not fit. It returns the rows and where each button is.
func (f *Form) drawButtons(labels []string, width, focus int) (rows [][]string, places []place) {
	var row []string
	used := 0
	flush := func() {
		if row != nil {
			rows = append(rows, row)
		}
		row, used = nil, 0
	}
	for i, label := range labels {
		text := widget.Clean(label)
		edge, body := widget.Faint, lipgloss.NewStyle()
		if label == f.submitted {
			text, edge, body = "✓ "+text, widget.Good, widget.Good
		}
		text = widget.Truncate(text, max(width-4, 1))
		b := thin
		if i == focus {
			b, edge, body = heavy, widget.Bold, widget.Selected
		}
		button := boxed([]string{text}, widget.Width(text), b, edge, body)
		w := widget.Width(text) + 4
		if used > 0 && used+1+w > width {
			flush()
		}
		if row == nil {
			row = button
		} else {
			for j := range row {
				row[j] += " " + button[j]
			}
			used++
		}
		places = append(places, place{row: len(rows), x0: used, x1: used + w})
		used += w
	}
	flush()
	return rows, places
}

// Update implements widget.Widget.
func (f *Form) Update(key string) (widget.Widget, widget.Result) {
	var res widget.Result
	f.clamp()
	switch key {
	case "j", "down", "l", "right":
		f.cursor++
	case "k", "up", "h", "left":
		f.cursor--
	case "space", "enter":
		c := f.controls[f.cursor]
		switch l := &f.lines[c.line]; l.kind {
		case option:
			res.Op = Choose{Text: l.text, Nth: f.nth(c), On: !l.on}
			if !l.on && l.radio {
				for i := range f.lines {
					if o := &f.lines[i]; o.kind == option && o.question == l.question {
						o.on = false
					}
				}
			}
			l.on = !l.on
			f.submitted = ""
		case field:
			nth, old := f.nth(c), l.text
			res.Prompt = &widget.Prompt{
				Label:   widget.Clean(f.questions[l.question]),
				Initial: old,
				Empty:   true,
				Submit:  func(text string) doc.Op { return Fill{Nth: nth, Old: old, Text: text} },
			}
		case buttons:
			f.submitted, f.at = l.labels[c.button], now().Format(time.RFC3339)
			res.Op = Submit{Label: f.submitted, At: f.at}
		}
	}
	f.clamp()
	return f, res
}

// Click implements widget.Clicker: a press on a control does what enter
// does there.
func (f *Form) Click(line, col int) (widget.Widget, widget.Result, bool) {
	for i, a := range f.drawn {
		if i < len(f.controls) && line >= a.lines.Start && line < a.lines.End && col >= a.x0 && col < a.x1 {
			f.cursor = i
			w, res := f.Update("enter")
			return w, res, true
		}
	}
	return f, widget.Result{}, false
}

// Sync implements widget.Widget.
func (f *Form) Sync(d doc.Document) widget.Widget {
	nf := parse(d, f.render)
	nf.cursor = f.cursor
	nf.clamp()
	return nf
}

// sentRule heads the answers of a sent form: the button that was pressed
// and when.
func (f *Form) sentRule(width int) string {
	label := widget.Clean(f.submitted)
	if t, ok := when.Parse(f.at); ok {
		label += " · " + widget.T("sent") + " " + when.Short(t, now())
	}
	rule := widget.Faint.Render("─ ") + widget.Good.Render("✓ ") + widget.Faint.Render(label+" ")
	return widget.Truncate(rule+widget.Faint.Render(strings.Repeat("─", max(width-widget.Width(rule), 0))), width)
}
