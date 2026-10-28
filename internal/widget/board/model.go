package board

import (
	"fmt"
	"strings"
	"time"

	"github.com/LeeSwallow/stickypane/internal/doc"
	"github.com/LeeSwallow/stickypane/internal/when"
	"github.com/LeeSwallow/stickypane/internal/widget"
)

type card struct {
	text   string
	who    []string  // who has it: the @names on its line
	moved  time.Time // when it last moved, as its line says
	detail []string
}

type column struct {
	title string
	cards []card
}

// content is what a board note says: the reader's side, with no screen
// state in it.
type content struct {
	desc []string // the lines before the first column
	cols []column
}

func parse(d doc.Document, e widget.Env) *Board { return &Board{content: read(d), env: e} }

// read reads a board note.
func read(d doc.Document) content {
	lines := doc.Lines(d.Body)
	spans := scan(lines)
	var b content
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
			text, who, moved := splitCard(lineText(lines[cs.start]))
			cd := card{text: text, who: who, moved: moved}
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
		return widget.T("1 card")
	}
	return fmt.Sprintf(widget.T("%d cards"), n)
}

// cardJSON is a card as a screen other than the terminal reads it.
type cardJSON struct {
	Text    string   `json:"text"`
	Who     []string `json:"who"`
	Moved   string   `json:"moved,omitempty"` // RFC 3339, when the line says
	Detail  []string `json:"detail"`
	Stalled bool     `json:"stalled"`
}

type columnJSON struct {
	Title string     `json:"title"`
	Cards []cardJSON `json:"cards"`
}

// data is the columns and their cards, from the same read the board draws,
// with the same call on which cards stalled.
func data(d doc.Document, _ widget.Env) any {
	b := read(d)
	t := now()
	out := struct {
		Desc    []string     `json:"desc"`
		Columns []columnJSON `json:"columns"`
	}{Desc: []string{}, Columns: []columnJSON{}}
	out.Desc = append(out.Desc, b.desc...)
	for ci, c := range b.cols {
		col := columnJSON{Title: c.title, Cards: []cardJSON{}}
		for _, cd := range c.cards {
			j := cardJSON{Text: cd.text, Who: []string{}, Detail: []string{}, Stalled: stalled(cd, ci == len(b.cols)-1, t)}
			j.Who = append(j.Who, cd.who...)
			j.Detail = append(j.Detail, cd.detail...)
			if !cd.moved.IsZero() {
				j.Moved = cd.moved.Format(when.Moment)
			}
			col.Cards = append(col.Cards, j)
		}
		out.Columns = append(out.Columns, col)
	}
	return out
}
