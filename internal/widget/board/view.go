// Package board is the kanban note: "## " headings are columns and the
// lines below them are cards, usually list items.
package board

import (
	"fmt"
	"strings"
	"time"

	"github.com/LeeSwallow/stickypane/internal/when"

	"github.com/LeeSwallow/stickypane/internal/doc"
	"github.com/LeeSwallow/stickypane/internal/widget"
)

const (
	minCol     = 14 // narrower than this, columns are stacked instead of side by side
	gutter     = 2  // empty cells between two columns
	formatHint = `no columns yet: add "## Name" headings`
)

// Board is the widget for a kanban note: the content and where the cursor
// is.
type Board struct {
	content
	col, row int    // cursor
	drawn    []area // where each card was in the last Draw
}

func (b *Board) clamp() {
	b.col = max(min(b.col, len(b.cols)-1), 0)
	n := 0
	if len(b.cols) > 0 {
		n = len(b.cols[b.col].cards)
	}
	b.row = max(min(b.row, n-1), 0)
}

func (b *Board) current() (card, bool) {
	if b.col < len(b.cols) && b.row < len(b.cols[b.col].cards) {
		return b.cols[b.col].cards[b.row], true
	}
	return card{}, false
}

// nth is how many cards above the cursor share the selected card's text. It
// tells an Op which of several identical cards is meant.
func (b *Board) nth() int {
	cards := b.cols[b.col].cards
	n := 0
	for _, c := range cards[:b.row] {
		if c.text == cards[b.row].text {
			n++
		}
	}
	return n
}

// wrapped returns s cleaned and wrapped to width, each line styled.
func wrapped(s string, width int, style func(...string) string) []string {
	lines := widget.Wrap(widget.Clean(s), max(width, 1))
	if style != nil {
		for i, l := range lines {
			lines[i] = style(l)
		}
	}
	return lines
}

// Draw implements widget.Widget. Every card is shown and long cards wrap.
// The selected card, marked only while the board is active, shows its
// details right below it.
func (b *Board) Draw(width int, active bool) (string, widget.Span) {
	var out []string
	for _, d := range b.desc {
		out = append(out, wrapped(d, width, widget.Faint.Render)...)
	}
	if len(b.cols) == 0 {
		hint := widget.Faint.Render(widget.Truncate(widget.T(formatHint), width))
		lines := []string{hint}
		for _, d := range b.desc {
			lines = append(lines, wrapped(d, width, nil)...)
		}
		return widget.Fit(strings.Join(lines, "\n"), width), widget.NoSpan
	}
	if len(out) > 0 {
		out = append(out, "")
	}
	b.clamp()

	cw := width / len(b.cols)
	var body []string
	at := widget.NoSpan
	b.drawn = b.drawn[:0]
	// place records where the cards of column i are: its lines start at
	// line top of the body and it spans the cells x0 to x1.
	place := func(i, top, x0, x1 int, cards []widget.Span) {
		for r, s := range cards {
			b.drawn = append(b.drawn, area{col: i, row: r, lines: s.Shift(len(out) + top), x0: x0, x1: x1})
		}
	}
	if cw < minCol {
		for i := range b.cols {
			if i > 0 {
				body = append(body, "")
			}
			lines, sel, cards := b.column(i, width-1, active)
			if sel.Ok() {
				at = sel.Shift(len(body))
			}
			place(i, len(body), 0, width, cards)
			body = append(body, lines...)
		}
	} else {
		cells := make([][]string, len(b.cols))
		rows := 0
		for i := range b.cols {
			var sel widget.Span
			var cards []widget.Span
			cells[i], sel, cards = b.column(i, cw-gutter, active)
			if sel.Ok() {
				at = sel
			}
			place(i, 0, i*cw, (i+1)*cw, cards)
			rows = max(rows, len(cells[i]))
		}
		body = joinColumns(cells, rows, cw)
	}
	return widget.Fit(strings.Join(append(out, body...), "\n"), width), at.Shift(len(out))
}

// area is where a card was in the last Draw.
type area struct {
	col, row int
	lines    widget.Span
	x0, x1   int
}

// Click implements widget.Clicker: a press on a card selects it.
func (b *Board) Click(line, col int) (widget.Widget, widget.Result, bool) {
	for _, a := range b.drawn {
		if line >= a.lines.Start && line < a.lines.End && col >= a.x0 && col < a.x1 {
			b.col, b.row = a.col, a.row
			b.clamp()
			return b, widget.Result{}, true
		}
	}
	return b, widget.Result{}, false
}

// boxWidth is the narrowest column whose cards are drawn as boxes; a
// narrower one draws them as lines with a bar.
const boxWidth = 12

// mark is the sign before a column's title: how far along its cards are,
// read from where the column stands, first to last.
func mark(i, n int) string {
	switch {
	case n > 1 && i == n-1:
		return widget.Good.Render("●")
	case i == 0:
		return widget.Faint.Render("○")
	}
	return widget.Info.Render("◐")
}

// column draws one column at width: its heading, then every card. Cards
// are boxes when there is room, lines with a bar when not; the selected
// card stands out and shows its details. The last column, where work ends,
// steps back. It returns the lines of the selected card and its details.
// When the selected column is empty that is its heading, so the screen
// still shows where a new card would go. A column that does not hold the
// selection, or a board that is not active, returns NoSpan. cards holds
// the lines of every card, details included.
func (b *Board) column(i, width int, active bool) (lines []string, at widget.Span, cards []widget.Span) {
	c := b.cols[i]
	at = widget.NoSpan
	if active && i == b.col && len(c.cards) == 0 {
		at = widget.Span{Start: 0, End: 2}
	}
	here := active && i == b.col
	count := widget.Faint.Render(fmt.Sprintf("(%d)", len(c.cards)))
	title := widget.Truncate(widget.Clean(c.title), max(width-3-widget.Width(count), 1))
	if here {
		title = widget.Bold.Underline(true).Render(title)
	} else {
		title = widget.Bold.Render(title)
	}
	lines = append(lines, mark(i, len(b.cols))+" "+title+" "+count)
	rule := widget.Faint
	if here {
		rule = widget.Accent
	}
	lines = append(lines, rule.Render(strings.Repeat("─", max(width, 1))))
	if len(c.cards) == 0 {
		lines = append(lines, widget.Faint.Render(widget.Truncate(widget.T("empty"), width)))
	}
	last := len(b.cols) > 1 && i == len(b.cols)-1
	for r, cd := range c.cards {
		selected := here && r == b.row
		from := len(lines)
		if width >= boxWidth {
			lines = append(lines, b.box(cd, width, selected, last)...)
		} else {
			if r > 0 {
				lines = append(lines, "")
				from++
			}
			lines = append(lines, b.bar(cd, width, selected, last)...)
		}
		if selected {
			at = widget.Span{Start: from, End: len(lines)}
		}
		cards = append(cards, widget.Span{Start: from, End: len(lines)})
	}
	return lines, at, cards
}

// box draws a card as a rounded box: its text, and its details when it is
// selected. The selected card's border takes the accent; a card in the
// last column is drawn faint.
func (b *Board) box(cd card, width int, selected, last bool) []string {
	inner := width - 4
	edge, text := widget.Faint, func(s string) string { return s }
	if last {
		text = func(s string) string { return widget.Faint.Render(s) }
	}
	if selected {
		edge, text = widget.Accent, func(s string) string { return widget.Bold.Render(s) }
	}
	row := func(s string) string {
		return edge.Render("│") + " " + s + strings.Repeat(" ", max(inner-widget.Width(s), 0)) + " " + edge.Render("│")
	}
	out := []string{edge.Render("╭" + strings.Repeat("─", inner+2) + "╮")}
	if !selected {
		meta := b.meta(cd, last, inner)
		for _, l := range wrapped(widget.Unlink(cd.text), inner, nil) {
			out = append(out, row(text(l)))
		}
		if meta != "" {
			out = append(out, row(meta))
		}
		return append(out, edge.Render("╰"+strings.Repeat("─", inner+2)+"╯"))
	}
	// The selected card says so in its text too, not by color alone, and
	// shows its details under its text.
	for j, l := range wrapped(widget.Unlink(cd.text), inner-2, nil) {
		lead := "  "
		if j == 0 {
			lead = "› "
		}
		out = append(out, row(lead+text(l)))
	}
	if meta := b.meta(cd, last, inner-2); meta != "" {
		out = append(out, row("  "+meta))
	}
	for _, d := range cd.detail {
		for _, l := range wrapped(d, inner-2, widget.Faint.Render) {
			out = append(out, row("  "+l))
		}
	}
	return append(out, edge.Render("╰"+strings.Repeat("─", inner+2)+"╯"))
}

// bar draws a card in a narrow column: a line with a bar down its left
// edge, or the cursor and its details when it is selected.
func (b *Board) bar(cd card, width int, selected bool, last bool) []string {
	var out []string
	for j, l := range wrapped(widget.Unlink(cd.text), width-2, nil) {
		switch {
		case selected && j == 0:
			out = append(out, widget.Selected.Render("› "+l))
		case selected:
			out = append(out, widget.Selected.Render("  "+l))
		default:
			out = append(out, widget.Faint.Render("▎")+" "+l)
		}
	}
	if meta := b.meta(cd, last, max(width-2, 1)); meta != "" {
		lead := widget.Faint.Render("▎") + " "
		if selected {
			lead = "  "
		}
		out = append(out, lead+meta)
	}
	if selected {
		for _, d := range cd.detail {
			for _, l := range wrapped(d, width-4, widget.Faint.Render) {
				out = append(out, "    "+l)
			}
		}
	}
	return out
}

// joinColumns lays cells out side by side, each column cw cells wide.
func joinColumns(cells [][]string, rows, cw int) []string {
	out := make([]string, rows)
	for r := range out {
		var sb strings.Builder
		for _, col := range cells {
			s := ""
			if r < len(col) {
				s = col[r]
			}
			sb.WriteString(widget.Pad(s, cw))
		}
		out[r] = strings.TrimRight(sb.String(), " ")
	}
	return out
}

// Update implements widget.Widget.
func (b *Board) Update(key string) (widget.Widget, widget.Result) {
	var res widget.Result
	if len(b.cols) == 0 {
		return b, res
	}
	b.clamp()
	col := b.cols[b.col]
	switch key {
	case "h", "left":
		b.col--
	case "l", "right":
		b.col++
	case "j", "down":
		b.row++
	case "k", "up":
		b.row--
	case "H", "L":
		to := b.col - 1
		if key == "L" {
			to = b.col + 1
		}
		if c, ok := b.current(); ok && to >= 0 && to < len(b.cols) {
			res.Op = MoveCard{From: col.title, To: b.cols[to].title, Text: c.text, Nth: b.nth()}
			// The card lands at the end of the target column. Point the
			// cursor there now; the next Sync makes that position real.
			b.col, b.row = to, len(b.cols[to].cards)
			return b, res
		}
	case "J", "K":
		delta := 1
		if key == "K" {
			delta = -1
		}
		if c, ok := b.current(); ok && b.row+delta >= 0 && b.row+delta < len(col.cards) {
			res.Op = ReorderCard{Col: col.title, Text: c.text, Nth: b.nth(), Delta: delta}
			b.row += delta
		}
	case "n":
		title := col.title
		res.Prompt = &widget.Prompt{
			Label:  "New card in " + widget.Clean(title),
			Submit: func(text string) doc.Op { return AddCard{Col: title, Text: text} },
		}
	}
	b.clamp()
	return b, res
}

// Sync implements widget.Widget.
func (b *Board) Sync(d doc.Document) widget.Widget {
	nb := parse(d)
	nb.col, nb.row = b.col, b.row
	nb.clamp()
	return nb
}

// now is the clock that says how long a card has not moved, replaced in
// tests.
var now = func() time.Time { return widget.Now() }

// StaleAfter is how long a card may stay where it is, outside the last
// column, before the board flags it; zero never flags. The settings set it.
var StaleAfter = 30 * time.Minute

// stalled reports whether a card has not moved for staleAfter.
func stalled(cd card, last bool, t time.Time) bool {
	return StaleAfter > 0 && !last && !cd.moved.IsZero() && t.Sub(cd.moved) >= StaleAfter
}

// meta is the line under a card: who has it, each name in a color of its
// own, when it moved, and ⚠ when it stalled. Empty when the card says
// neither.
//
// The line fits width: when it is too long the names are cut, never the
// time or the flag, which say whether to look.
func (b *Board) meta(cd card, last bool, width int) string {
	t := now()
	tail := ""
	if s := when.Short(cd.moved, t); s != "" {
		tail = widget.Faint.Render(s)
	}
	if stalled(cd, last, t) {
		tail += " " + widget.Warn.Render("⚠")
	}
	var names []string
	for _, w := range cd.who {
		names = append(names, widget.NameStyle(w).Render("@"+widget.Clean(w)))
	}
	head := strings.Join(names, " ")
	switch {
	case head == "":
		return widget.Truncate(tail, width)
	case tail == "":
		return widget.Truncate(head, width)
	}
	sep := widget.Faint.Render(" · ")
	room := width - widget.Width(tail) - widget.Width(sep)
	if room < 4 {
		return widget.Truncate(tail, width)
	}
	return widget.Truncate(head, room) + sep + tail
}

// NextChange implements widget.Timed: when the next card stalls.
func (b *Board) NextChange(t time.Time) time.Time {
	var next time.Time
	for i, c := range b.cols {
		last := len(b.cols) > 1 && i == len(b.cols)-1
		for _, cd := range c.cards {
			if last || cd.moved.IsZero() || StaleAfter <= 0 {
				continue
			}
			if at := cd.moved.Add(StaleAfter); at.After(t) && (next.IsZero() || at.Before(next)) {
				next = at
			}
		}
	}
	return next
}
