// Package board is the kanban note: "## " headings are columns and the
// lines below them are cards, usually list items.
package board

import (
	"fmt"
	"strings"

	"github.com/LeeSwallow/stickypane/internal/doc"
	"github.com/LeeSwallow/stickypane/internal/widget"
)

const (
	minCol     = 14 // narrower than this, columns are stacked instead of side by side
	gutter     = 2  // empty cells between two columns
	formatHint = `no columns yet: add "## Name" headings`
)

// Kind registers the board.
var Kind = widget.Kind{
	Name:    "board",
	Label:   "Board (kanban)",
	Icon:    "▦",
	Hint:    []string{"h l j k", "move", "H L", "shift card", "J K", "reorder", "n", "new card"},
	Blurb:   "Cards in columns. Move a card as the work moves.",
	Example: "## To do\n- payments\n## Doing\n- login API\n## Done\n- schema\n",
	Keys:    "h l j k H L J K n left right up down",
	Size:    func(doc.Document) string { return widget.SizePage },
	Template: func(title string) []byte {
		return widget.NewFile("board", title, "## To do\n\n## Doing\n\n## Done\n")
	},
	Parse: func(d doc.Document) widget.Widget { return parse(d) },
}

type card struct {
	text   string
	detail []string
}

type column struct {
	title string
	cards []card
}

// Board is the widget for a kanban note.
type Board struct {
	desc     []string // the lines before the first column
	cols     []column
	col, row int    // cursor
	drawn    []area // where each card was in the last Draw
}

func parse(d doc.Document) *Board {
	lines := doc.Lines(d.Body)
	spans := scan(lines)
	b := &Board{}
	before := len(lines)
	if len(spans) > 0 {
		before = spans[0].head
	}
	for _, l := range lines[:before] {
		if !isBlank(l) {
			b.desc = append(b.desc, strings.TrimSpace(l))
		}
	}
	for _, s := range spans {
		c := column{title: s.title}
		for _, cs := range s.cards {
			cd := card{text: cs.text}
			for _, l := range lines[cs.start+1 : cs.end] {
				if !isBlank(l) {
					// A detail is often a nested list item; its marker is noise here.
					cd.detail = append(cd.detail, cardText(strings.TrimSpace(l)))
				}
			}
			c.cards = append(c.cards, cd)
		}
		b.cols = append(b.cols, c)
	}
	return b
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
		hint := widget.Faint.Render(widget.Truncate(formatHint, width))
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

// column draws one column at width: its heading, then every card with a
// blank line between cards. It returns the lines of the selected card and
// its details. When the selected column is empty that is its heading, so
// the screen still shows where a new card would go. A column that does not
// hold the selection, or a board that is not active, returns NoSpan. cards
// holds the lines of every card, details included.
func (b *Board) column(i, width int, active bool) (lines []string, at widget.Span, cards []widget.Span) {
	c := b.cols[i]
	at = widget.NoSpan
	if active && i == b.col && len(c.cards) == 0 {
		at = widget.Span{Start: 0, End: 2}
	}
	head := widget.Truncate(fmt.Sprintf("%s (%d)", widget.Clean(c.title), len(c.cards)), width)
	if active && i == b.col {
		lines = append(lines, widget.Bold.Underline(true).Render(head))
	} else {
		lines = append(lines, widget.Bold.Render(head))
	}
	lines = append(lines, widget.Faint.Render(strings.Repeat("─", max(width, 1))))
	for r, cd := range c.cards {
		if r > 0 {
			lines = append(lines, "")
		}
		selected := active && i == b.col && r == b.row
		from := len(lines)
		for j, l := range wrapped(cd.text, width-2, nil) {
			switch {
			case selected && j == 0:
				at.Start = len(lines)
				lines = append(lines, widget.Selected.Render("› "+l))
			case selected:
				lines = append(lines, widget.Selected.Render("  "+l))
			default:
				// The bar down the left edge is what makes it read as a card.
				lines = append(lines, widget.Faint.Render("▎")+" "+l)
			}
		}
		if selected {
			for _, d := range cd.detail {
				for _, l := range wrapped(d, width-4, widget.Faint.Render) {
					lines = append(lines, "    "+l)
				}
			}
			at.End = len(lines)
		}
		cards = append(cards, widget.Span{Start: from, End: len(lines)})
	}
	return lines, at, cards
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

// Summary implements widget.Widget: how many cards the board holds.
func (b *Board) Summary() string {
	n := 0
	for _, c := range b.cols {
		n += len(c.cards)
	}
	switch {
	case len(b.cols) == 0:
		return ""
	case n == 1:
		return "1 card"
	}
	return fmt.Sprintf("%d cards", n)
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
