package board

import (
	"fmt"
	"strings"

	"github.com/LeeSwallow/stickypane/internal/doc"
	"github.com/LeeSwallow/stickypane/internal/widget"
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
	Template: func(title string) []byte {
		return widget.NewFile("board", title, "## To do\n\n## Doing\n\n## Done\n")
	},
	Options: []widget.Option{
		{Name: "columns", Type: widget.OptLines, Label: "Columns", Default: "To do\nDoing\nDone"},
		{Name: "cards", Type: widget.OptLines, Label: "First cards"},
	},
	Compose: compose,
	Parse:   func(d doc.Document) widget.Widget { return parse(d) },
	Events:  events,
	Data:    data,
	Act:     act,
	Tend:    tend,
}

// compose writes a new board: a "## " heading for each column, and the
// first cards under them. A card goes to the first column, or to the one
// it names in front of it ("Doing: login API"); an @name in it stays, as
// who has it.
func compose(title string, opts map[string]string) ([]byte, error) {
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
