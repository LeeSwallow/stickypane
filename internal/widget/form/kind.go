package form

import (
	"github.com/LeeSwallow/stickypane/internal/doc"
	"github.com/LeeSwallow/stickypane/internal/widget"
	"github.com/LeeSwallow/stickypane/internal/widget/note"
)

// NewKind returns the form kind. Its text is drawn with render, the same
// way a plain note is.
func NewKind(render note.Renderer) widget.Kind {
	return widget.Kind{
		Name:    "form",
		Label:   "Form",
		Icon:    "◉",
		Hint:    []string{"j k", "move", "enter", "choose or press"},
		Blurb:   "A document that asks: options to choose, lines to fill in, buttons to press. The answers land in the file.",
		Command: "stickypane wait deploy --timeout 10m",
		Usage:   "\"- ( )\" options are one choice, \"- [ ]\" options many, \"> \" lines are filled in and \"[ Label ]\" lines are buttons. Pressing a button writes \"submitted: Label\" into the front matter; `stickypane wait` returns the answers then.",
		Example: "## Where to deploy?\n- (x) staging\n- ( ) production\n\n## Note\n> after lunch\n\n[ Deploy ] [ Cancel ]\n",
		Keys:    "j k up down h l left right space enter",
		Size:    func(doc.Document) string { return widget.SizeHalf },
		Template: func(title string) []byte {
			return widget.NewFile("form", title, "## Question\n- ( ) Yes\n- ( ) No\n\n[ "+defaultButton+" ]\n")
		},
		Parse:  func(d doc.Document) widget.Widget { return parse(d, render) },
		Events: events,
	}
}
