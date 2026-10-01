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
	minCol     = 16 // narrower than this, columns are stacked instead of side by side
	formatHint = `no columns yet: add "## Name" headings`
)

// Kind registers the board.
var Kind = widget.Kind{
	Name:  "board",
	Label: "Board (kanban)",
	Keys:  "h l j k H L J K n left right up down",
	Size:  func(doc.Document) string { return widget.SizePage },
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
	col, row int // cursor
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
func (b *Board) Draw(width int, active bool) (string, int) {
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
		return widget.Fit(strings.Join(lines, "\n"), width), -1
	}
	if len(out) > 0 {
		out = append(out, "")
	}
	b.clamp()

	cw := width / len(b.cols)
	var body []string
	cursor := -1
	if cw < minCol {
		for i := range b.cols {
			if i > 0 {
				body = append(body, "")
			}
			lines, at := b.column(i, width, active)
			if at >= 0 {
				cursor = len(body) + at
			}
			body = append(body, lines...)
		}
	} else {
		cells := make([][]string, len(b.cols))
		rows := 0
		for i := range b.cols {
			var at int
			cells[i], at = b.column(i, cw-1, active)
			if at >= 0 {
				cursor = at
			}
			rows = max(rows, len(cells[i]))
		}
		body = joinColumns(cells, rows, cw)
	}
	if cursor >= 0 {
		cursor += len(out)
	}
	return widget.Fit(strings.Join(append(out, body...), "\n"), width), cursor
}

// column draws one column at width: its heading, then every card with a
// blank line between cards. It returns the line of the selected card, or -1
// when the selection is in another column or the board is not active.
func (b *Board) column(i, width int, active bool) (lines []string, cursor int) {
	c := b.cols[i]
	cursor = -1
	head := widget.Truncate(fmt.Sprintf("%s (%d)", widget.Clean(c.title), len(c.cards)), width)
	if active && i == b.col {
		lines = append(lines, widget.Bold.Underline(true).Render(head))
	} else {
		lines = append(lines, widget.Bold.Render(head))
	}
	for r, cd := range c.cards {
		if r > 0 {
			lines = append(lines, "")
		}
		selected := active && i == b.col && r == b.row
		for j, l := range wrapped(cd.text, width-2, nil) {
			switch {
			case selected && j == 0:
				cursor = len(lines)
				lines = append(lines, widget.Selected.Render("› "+l))
			case selected:
				lines = append(lines, widget.Selected.Render("  "+l))
			default:
				lines = append(lines, "  "+l)
			}
		}
		if selected {
			for _, d := range cd.detail {
				for _, l := range wrapped(d, width-4, widget.Faint.Render) {
					lines = append(lines, "    "+l)
				}
			}
		}
	}
	return lines, cursor
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
