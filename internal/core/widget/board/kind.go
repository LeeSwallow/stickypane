package board

import (
	"errors"
	"fmt"
	"strings"

	"github.com/LeeSwallow/stickypane/internal/core/doc"
	"github.com/LeeSwallow/stickypane/internal/core/widget"
)

// Kind registers the board.
var Kind = widget.Kind{
	Name:    "board",
	Label:   "Board (kanban)",
	Icon:    "▦",
	Hint:    []string{"h l j k", "move", "H L", "shift card", "J K", "reorder", "n", "new card"},
	Blurb:   "Cards in columns. Move a card as the work moves.",
	Command: "stickypane card work add \"login API\" --to Doing",
	Usage:   "Each \"## Heading\" is a column and each top-level \"- item\" a card; indented lines are its details. A card may end with who has it, \"@codex-1\"; the board writes when a card moved (\"@{2026-10-03} @@{14:02}\") and flags one that has not moved for half an hour outside the last column. The user moves cards with keys; `stickypane card work move login --to Done` moves one from your side.",
	Example: "## To do\n- payments\n## Doing\n- login API\n## Done\n- schema\n",
	Keys:    "h l j k H L J K n left right up down",
	Size:    func(doc.Document) string { return widget.SizeFull },
	Template: func(title string, e widget.Env) []byte {
		return start(title, Columns(e))
	},
	Options: []widget.Option{
		{Name: "columns", Type: widget.OptLines, Label: "Columns", Default: strings.Join(columns, "\n"), Words: true},
		{Name: "cards", Type: widget.OptLines, Label: "First cards"},
	},
	Compose: compose,
	Parse:   func(d doc.Document, e widget.Env) widget.Widget { return parse(d, e) },
	Events:  events,
	Data:    data,
	Act:     act,
	Tend:    tend,
}

// columns are the columns a new board starts with, in English. Each is a
// phrase of the "labels" of a language file, so a board made in another
// language starts with them in its words (Columns).
var columns = []string{"To do", "Doing", "Done"}

// Columns are the columns a new board starts with, in e's words.
func Columns(e widget.Env) []string {
	out := make([]string, len(columns))
	for i, c := range columns {
		out[i] = e.T(c)
	}
	return out
}

// Starting is a new board called title for a first card that goes to
// column to: the board of Template when to is empty or names one of its
// columns (as Add names one); the usual columns in English when to names
// one of those, as an agent that works in English does on a board of
// another language; else a board of that one column alone, so a card put
// in a column of its own leaves no empty columns beside it.
func Starting(title, to string, e widget.Env) []byte {
	if to = strings.TrimSpace(to); to == "" {
		return start(title, Columns(e))
	}
	for _, cols := range [][]string{Columns(e), columns} {
		if _, err := widget.Pick(cols, to, "column"); !errors.Is(err, widget.ErrNoMatch) {
			return start(title, cols)
		}
	}
	return start(title, []string{to})
}

// start is a new board called title with the columns titles and no cards.
func start(title string, titles []string) []byte {
	b, _ := compose(title, map[string]string{"columns": strings.Join(titles, "\n")}, widget.Env{})
	return b
}

// compose writes a new board: a "## " heading for each column, and the
// first cards under them. A card goes to the first column, or to the one
// it names in front of it ("Doing: login API"); an @name in it stays, as
// who has it.
func compose(title string, opts map[string]string, _ widget.Env) ([]byte, error) {
	titles := widget.Lines(opts["columns"])
	if len(titles) == 0 {
		return nil, fmt.Errorf("%w: a board needs a column", widget.ErrBadOption)
	}
	cards := make([][]string, len(titles))
	for _, card := range widget.Lines(opts["cards"]) {
		col := 0
		if name, text, ok := strings.Cut(card, ":"); ok {
			for i, t := range titles {
				if strings.EqualFold(strings.TrimSpace(name), t) && strings.TrimSpace(text) != "" {
					col, card = i, strings.TrimSpace(text)
					break
				}
			}
		}
		cards[col] = append(cards[col], card)
	}
	var b strings.Builder
	for i, t := range titles {
		if i > 0 {
			b.WriteString("\n")
		}
		b.WriteString("## " + t + "\n")
		for _, c := range cards[i] {
			b.WriteString("- " + c + "\n")
		}
	}
	return widget.NewFile("board", title, b.String()), nil
}
