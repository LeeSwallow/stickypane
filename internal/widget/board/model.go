package board

import (
	"fmt"
	"strings"
	"time"

	"github.com/LeeSwallow/stickypane/internal/doc"
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

func parse(d doc.Document) *Board { return &Board{content: read(d)} }

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
