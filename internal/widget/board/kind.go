package board

import (
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
	Usage:   "Each \"## Heading\" is a column and each top-level \"- item\" a card; indented lines are its details. The user moves cards with keys and the file changes; `stickypane card work move login --to Done` moves one from your side.",
	Example: "## To do\n- payments\n## Doing\n- login API\n## Done\n- schema\n",
	Keys:    "h l j k H L J K n left right up down",
	Size:    func(doc.Document) string { return widget.SizePage },
	Template: func(title string) []byte {
		return widget.NewFile("board", title, "## To do\n\n## Doing\n\n## Done\n")
	},
	Parse: func(d doc.Document) widget.Widget { return parse(d) },
}
