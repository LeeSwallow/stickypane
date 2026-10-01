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
	previewCards   = 5  // cards shown per column on the board
	stackedCards   = 3  // cards shown per column when columns are stacked
	minPreviewCol  = 12 // narrower than this, the preview stacks columns
	minModalCol    = 16 // narrower than this, the open board scrolls sideways
	detailRows     = 4  // separator plus three lines about the selected card
	descRows       = 2  // lines of the board's description shown when open
	minDetailModal = 8  // shorter than this, the open board shows cards only
	formatHint     = `no columns yet: add "## Name" headings`
)

// Kind registers the board.
var Kind = widget.Kind{
	Name:    "board",
	Label:   "Board (kanban)",
	FullRow: true,
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
					cd.detail = append(cd.detail, strings.TrimSpace(l))
				}
			}
			c.cards = append(c.cards, cd)
		}
		b.cols = append(b.cols, c)
	}
	return b
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

// unshaped draws a board that has no columns: the format hint followed by
// the text that is there, so a note that does not fit the shape stays visible.
func (b *Board) unshaped(width, height int) string {
	lines := []string{widget.Faint.Render(widget.Truncate(formatHint, width))}
	for _, d := range b.desc {
		lines = append(lines, widget.Truncate(widget.Clean(d), width))
	}
	return strings.Join(widget.Window(lines, 0, height), "\n")
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

func header(c column) string {
	return fmt.Sprintf("%s (%d)", widget.Clean(c.title), len(c.cards))
}

// Preview implements widget.Widget.
func (b *Board) Preview(width int) string {
	if len(b.cols) == 0 {
		return b.unshaped(width, previewCards)
	}
	cw := width / len(b.cols)
	if cw < minPreviewCol {
		return b.stacked(width)
	}
	cells := make([][]string, len(b.cols))
	rows := 0
	for i, c := range b.cols {
		cells[i] = []string{widget.Bold.Render(widget.Truncate(header(c), cw-1))}
		for j, cd := range c.cards {
			if j == previewCards {
				cells[i] = append(cells[i], widget.Faint.Render(fmt.Sprintf("+%d more", len(c.cards)-j)))
				break
			}
			cells[i] = append(cells[i], widget.Truncate(widget.Clean(cd.text), cw-1))
		}
		rows = max(rows, len(cells[i]))
	}
	return joinColumns(cells, rows, cw)
}

func (b *Board) stacked(width int) string {
	var out []string
	for _, c := range b.cols {
		out = append(out, widget.Bold.Render(widget.Truncate(header(c), width)))
		for j, cd := range c.cards {
			if j == stackedCards {
				out = append(out, widget.Faint.Render(fmt.Sprintf("  +%d more", len(c.cards)-j)))
				break
			}
			out = append(out, "  "+widget.Truncate(widget.Clean(cd.text), width-2))
		}
	}
	return strings.Join(out, "\n")
}

// joinColumns lays cells out side by side, each column cw cells wide.
func joinColumns(cells [][]string, rows, cw int) string {
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
	return strings.Join(out, "\n")
}

// View implements widget.Widget.
func (b *Board) View(width, height int) string {
	if len(b.cols) == 0 {
		return b.unshaped(width, height)
	}
	b.clamp()
	detail, desc := 0, 0
	if height >= minDetailModal {
		detail, desc = detailRows, min(len(b.desc), descRows)
	}
	listH := max(height-detail-desc-1, 1)

	cw, first, visible := width/len(b.cols), 0, len(b.cols)
	if cw < minModalCol {
		cw = max(min(minModalCol, width), 1)
		visible = max(width/cw, 1)
		first = max(b.col-visible+1, 0)
	}

	var cells [][]string
	for i := first; i < first+visible && i < len(b.cols); i++ {
		c := b.cols[i]
		head := widget.Bold.Render(widget.Truncate(header(c), cw-1))
		if i == b.col {
			head = widget.Bold.Underline(true).Render(widget.Truncate(header(c), cw-1))
		}
		lines := []string{head}
		offset := 0
		if i == b.col {
			offset = max(b.row-listH+1, 0)
		}
		for r := offset; r < len(c.cards) && r < offset+listH; r++ {
			text := widget.Truncate(widget.Clean(c.cards[r].text), cw-3)
			if i == b.col && r == b.row {
				lines = append(lines, widget.Selected.Render("› "+text))
			} else {
				lines = append(lines, "  "+text)
			}
		}
		cells = append(cells, lines)
	}
	var out []string
	for i, d := range b.desc[:desc] {
		if i == desc-1 && len(b.desc) > desc {
			d += " …"
		}
		out = append(out, widget.Faint.Render(widget.Truncate(widget.Clean(d), width)))
	}
	out = append(out, joinColumns(cells, listH+1, cw))
	if detail > 0 {
		out = append(out, widget.Faint.Render(strings.Repeat("─", width)))
		out = append(out, b.detail(width, detail-1)...)
	}
	return strings.Join(out, "\n")
}

// detail describes the selected card in at most n lines.
func (b *Board) detail(width, n int) []string {
	c, ok := b.current()
	if !ok {
		return []string{widget.Faint.Render("no card selected")}
	}
	lines := []string{widget.Bold.Render(widget.Truncate(widget.Clean(c.text), width))}
	for _, d := range c.detail {
		lines = append(lines, widget.Truncate(widget.Clean(d), width))
	}
	if len(c.detail) == 0 {
		lines = append(lines, widget.Faint.Render("(no details)"))
	}
	return lines[:min(len(lines), n)]
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
