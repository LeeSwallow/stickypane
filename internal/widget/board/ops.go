package board

import (
	"strings"

	"github.com/LeeSwallow/stickypane/internal/doc"
)

// cardSpan is a card's place in the body: its first line and the detail
// lines below it, as the half-open line range [start, end).
type cardSpan struct {
	text       string
	start, end int
}

// colSpan is a column: its "## " heading line and the cards below it.
type colSpan struct {
	title string
	head  int
	cards []cardSpan
}

func isHeading(line string) bool { return strings.HasPrefix(line, "## ") }

func isBlank(line string) bool { return strings.TrimSpace(line) == "" }

func isDetail(line string) bool {
	return (strings.HasPrefix(line, " ") || strings.HasPrefix(line, "\t")) && !isBlank(line)
}

// cardText is what identifies and labels a card: the line without its list
// marker. A line that is not a list item is a card too, so nothing written
// under a column is ever hidden.
func cardText(line string) string {
	if strings.HasPrefix(line, "- ") || strings.HasPrefix(line, "* ") {
		line = line[2:]
	}
	return strings.TrimSpace(line)
}

// scan finds the columns and cards in body lines. Every non-blank line under
// a column starts a card and owns the indented lines below it, including
// blank lines inside those details (such as in a code block). Lines before
// the first heading belong to no column.
func scan(lines []string) []colSpan {
	var cols []colSpan
	for i := 0; i < len(lines); i++ {
		line := lines[i]
		if isHeading(line) {
			cols = append(cols, colSpan{title: strings.TrimSpace(line[3:]), head: i})
			continue
		}
		if len(cols) == 0 || isBlank(line) {
			continue
		}
		c := cardSpan{text: cardText(line), start: i, end: i + 1}
		for j := c.end; j < len(lines); j++ {
			if isDetail(lines[j]) {
				c.end = j + 1
			} else if !isBlank(lines[j]) {
				break
			}
		}
		last := &cols[len(cols)-1]
		last.cards = append(last.cards, c)
		i = c.end - 1
	}
	return cols
}

func findCol(cols []colSpan, title string) int {
	for i, c := range cols {
		if c.title == title {
			return i
		}
	}
	return -1
}

// findCard returns the index of the nth card (from zero) with the given
// text, or -1 when the column has fewer such cards.
func findCard(col colSpan, text string, nth int) int {
	for i, c := range col.cards {
		if c.text != text {
			continue
		}
		if nth == 0 {
			return i
		}
		nth--
	}
	return -1
}

// insertAt is the line index where a new card goes: after the column's last
// card, or right below the heading when the column is empty.
func insertAt(col colSpan) int {
	if n := len(col.cards); n > 0 {
		return col.cards[n-1].end
	}
	return col.head + 1
}

func splice(lines []string, at int, block ...string) []string {
	out := make([]string, 0, len(lines)+len(block))
	out = append(out, lines[:at]...)
	out = append(out, block...)
	return append(out, lines[at:]...)
}

// MoveCard moves a card from column From to the end of column To. The card
// is the one with Text; when several cards in From share that text, Nth
// counts from zero to say which.
type MoveCard struct {
	From, To, Text string
	Nth            int
}

// Apply implements doc.Op.
func (o MoveCard) Apply(d doc.Document) (doc.Document, error) {
	lines := doc.Lines(d.Body)
	cols := scan(lines)
	from := findCol(cols, o.From)
	if from < 0 || findCol(cols, o.To) < 0 {
		return d, doc.ErrConflict
	}
	i := findCard(cols[from], o.Text, o.Nth)
	if i < 0 {
		return d, doc.ErrConflict
	}
	c := cols[from].cards[i]
	block := append([]string(nil), lines[c.start:c.end]...)
	rest := append(append([]string(nil), lines[:c.start]...), lines[c.end:]...)
	cols = scan(rest)
	d.Body = doc.Join(splice(rest, insertAt(cols[findCol(cols, o.To)]), block...))
	return d, nil
}

// ReorderCard moves a card up (Delta -1) or down (Delta 1) inside column
// Col. The card is the Nth one (from zero) with Text. At the edge of the
// column it changes nothing.
type ReorderCard struct {
	Col, Text string
	Nth       int
	Delta     int
}

// Apply implements doc.Op.
func (o ReorderCard) Apply(d doc.Document) (doc.Document, error) {
	lines := doc.Lines(d.Body)
	cols := scan(lines)
	ci := findCol(cols, o.Col)
	if ci < 0 {
		return d, doc.ErrConflict
	}
	cards := cols[ci].cards
	i := findCard(cols[ci], o.Text, o.Nth)
	if i < 0 {
		return d, doc.ErrConflict
	}
	j := i + o.Delta
	if j < 0 || j >= len(cards) {
		return d, nil
	}
	lo, hi := cards[min(i, j)], cards[max(i, j)]
	out := make([]string, 0, len(lines))
	out = append(out, lines[:lo.start]...)
	out = append(out, lines[hi.start:hi.end]...)
	out = append(out, lines[lo.end:hi.start]...)
	out = append(out, lines[lo.start:lo.end]...)
	out = append(out, lines[hi.end:]...)
	d.Body = doc.Join(out)
	return d, nil
}

// AddCard appends a card to the end of column Col.
type AddCard struct{ Col, Text string }

// Apply implements doc.Op.
func (o AddCard) Apply(d doc.Document) (doc.Document, error) {
	lines := doc.Lines(d.Body)
	cols := scan(lines)
	ci := findCol(cols, o.Col)
	if ci < 0 {
		return d, doc.ErrConflict
	}
	d.Body = doc.Join(splice(lines, insertAt(cols[ci]), "- "+o.Text+doc.EOL(d.Body)))
	return d, nil
}
