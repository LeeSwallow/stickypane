package board

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/LeeSwallow/stickypane/internal/core/doc"
	"github.com/LeeSwallow/stickypane/internal/core/widget"
)

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
	block[0] = unmoved(block[0]) // the board writes when it moved
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

// firstColumn is the column an empty board gets when a card is added.
const firstColumn = "To do"

// Move moves a card to the end of another column for a caller that names
// both instead of pointing at them: Card and To are the text, a part of it,
// or a position ("#2"; cards count across the whole board). A card already
// in that column stays where it is.
type Move struct{ Card, To string }

// Apply implements doc.Op.
func (o Move) Apply(d doc.Document) (doc.Document, error) {
	cols := scan(doc.Lines(d.Body))
	var titles, texts []string
	var owner []int
	for ci, c := range cols {
		titles = append(titles, c.title)
		for _, cd := range c.cards {
			texts = append(texts, cd.text)
			owner = append(owner, ci)
		}
	}
	n, err := widget.Pick(texts, o.Card, "card")
	if err != nil {
		return d, err
	}
	to, err := widget.Pick(titles, o.To, "column")
	if err != nil {
		return d, err
	}
	if owner[n] == to {
		return d, nil
	}
	nth := 0
	for i := 0; i < n; i++ {
		if owner[i] == owner[n] && texts[i] == texts[n] {
			nth++
		}
	}
	return MoveCard{From: titles[owner[n]], To: titles[to], Text: texts[n], Nth: nth}.Apply(d)
}

// Add puts a new card at the end of column To, named like Move names it.
// Without To the card goes to the first column. A column that does not
// exist yet is added at the end of the board.
type Add struct{ Card, To string }

// Apply implements doc.Op.
func (o Add) Apply(d doc.Document) (doc.Document, error) {
	cols := scan(doc.Lines(d.Body))
	var titles []string
	for _, c := range cols {
		titles = append(titles, c.title)
	}
	col := o.To
	switch {
	case col == "" && len(cols) > 0:
		col = titles[0]
	case col == "":
		col = firstColumn
	default:
		i, err := widget.Pick(titles, col, "column")
		switch {
		case err == nil:
			col = titles[i]
		case !errors.Is(err, widget.ErrNoMatch):
			return d, err
		}
	}
	if findCol(cols, col) < 0 {
		d.Body = doc.AppendLine(d.Body, "## "+strings.TrimSpace(col))
	}
	return AddCard{Col: strings.TrimSpace(col), Text: doc.OneLine(o.Card)}.Apply(d)
}

// StampMoved writes At onto the cards that moved, by name, and onto every
// card without a time yet.
type StampMoved struct {
	At    time.Time
	Moved map[string]bool
}

// Apply implements doc.Op.
func (o StampMoved) Apply(d doc.Document) (doc.Document, error) {
	lines := doc.Lines(d.Body)
	for _, c := range scan(lines) {
		for _, cs := range c.cards {
			_, _, moved := splitCard(lineText(lines[cs.start]))
			if moved.IsZero() || o.Moved[c.title+"\x00"+cs.text] {
				lines[cs.start] = movedAt(lines[cs.start], o.At)
			}
		}
	}
	d.Body = doc.Join(lines)
	return d, nil
}

// tend is the board's Kind.Tend: a card that appeared, moved to another
// column since before, or has no time yet, gets the time the file changed.
func tend(before, after doc.Document, changed time.Time) doc.Op {
	was := map[string]string{} // card name -> its column before
	for _, c := range scan(doc.Lines(before.Body)) {
		for _, cs := range c.cards {
			if _, ok := was[cs.text]; !ok {
				was[cs.text] = c.title
			}
		}
	}
	lines := doc.Lines(after.Body)
	moved := map[string]bool{}
	need := false
	for _, c := range scan(lines) {
		for _, cs := range c.cards {
			_, _, t := splitCard(lineText(lines[cs.start]))
			col, had := was[cs.text]
			switch {
			case t.IsZero():
				need = true
			case len(was) > 0 && (!had || col != c.title):
				moved[c.title+"\x00"+cs.text] = true
				need = true
			}
		}
	}
	if !need {
		return nil
	}
	return StampMoved{At: changed, Moved: moved}
}

// MoveCardAt moves a card to column To as MoveCard does, then up to place
// At (from zero) in it; At at or past the end leaves it last.
type MoveCardAt struct {
	From, To, Text string
	Nth, At        int
}

// Apply implements doc.Op.
func (o MoveCardAt) Apply(d doc.Document) (doc.Document, error) {
	d, err := MoveCard{From: o.From, To: o.To, Text: o.Text, Nth: o.Nth}.Apply(d)
	if err != nil {
		return d, err
	}
	cols := scan(doc.Lines(d.Body))
	col := cols[findCol(cols, o.To)]
	last := len(col.cards) - 1
	nth := 0 // among the cards with this text, the moved one is the last
	for _, c := range col.cards[:last] {
		if c.text == o.Text {
			nth++
		}
	}
	for at := last; at > o.At && at > 0; at-- {
		if d, err = (ReorderCard{Col: o.To, Text: o.Text, Nth: nth, Delta: -1}).Apply(d); err != nil {
			return d, err
		}
		// The card passed one above it; if that had the same text, it is
		// now one earlier among them.
		if col.cards[at-1].text == o.Text {
			nth--
		}
	}
	return d, nil
}

// act is the board's Kind.Act: "move" with col, i (the card's place in it,
// from zero), to (the column it goes to) and at (its place there, from
// zero; the end when the argument is absent, an error when it is not
// a number or is past the column), and "add" with col and text. Columns count
// from zero, in the order of the Data. When a card moved is not written
// here; tend writes it.
func act(d doc.Document, a widget.Action, _ widget.Env) (doc.Op, error) {
	cols := scan(doc.Lines(d.Body))
	index := func(verb, name string, n int) (int, error) {
		v, err := strconv.Atoi(a.Args[name])
		if err != nil {
			return 0, fmt.Errorf("%s: %s must be a number, not %q", verb, name, a.Args[name])
		}
		if v < 0 || v >= n {
			return 0, fmt.Errorf("%s: no %s %d, there are %d", verb, name, v, n)
		}
		return v, nil
	}
	switch a.Name {
	case "move":
		from, err := index("move", "col", len(cols))
		if err != nil {
			return nil, err
		}
		i, err := index("move", "i", len(cols[from].cards))
		if err != nil {
			return nil, err
		}
		to, err := index("move", "to", len(cols))
		if err != nil {
			return nil, err
		}
		room := len(cols[to].cards) // places in the column once the card is in it
		if from != to {
			room++
		}
		at := room - 1
		if _, ok := a.Args["at"]; ok {
			if at, err = index("move", "at", room); err != nil {
				return nil, err
			}
		}
		text := cols[from].cards[i].text
		nth := 0
		for _, c := range cols[from].cards[:i] {
			if c.text == text {
				nth++
			}
		}
		return MoveCardAt{From: cols[from].title, To: cols[to].title, Text: text, Nth: nth, At: at}, nil
	case "add":
		ci, err := index("add", "col", len(cols))
		if err != nil {
			return nil, err
		}
		text := strings.TrimSpace(a.Args["text"])
		if text == "" || strings.ContainsAny(text, "\r\n") {
			return nil, fmt.Errorf("add: text must be one line, not empty")
		}
		return AddCard{Col: cols[ci].title, Text: text}, nil
	}
	return nil, fmt.Errorf("%w: %q", widget.ErrUnknownAction, a.Name)
}
