package form

import (
	"fmt"
	"strings"

	"github.com/LeeSwallow/stickypane/internal/core/doc"
	"github.com/LeeSwallow/stickypane/internal/core/widget"
	"github.com/LeeSwallow/stickypane/internal/core/widget/note"
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
		Options: []widget.Option{
			{Name: "question", Type: widget.OptText, Label: "Question", Default: "Question"},
			{Name: "answer", Type: widget.OptChoice, Label: "Answer", Choices: []string{"one", "many", "text"}, Default: "one"},
			{Name: "choices", Type: widget.OptLines, Label: "Choices", Default: "Yes\nNo"},
			{Name: "buttons", Type: widget.OptLines, Label: "Buttons", Default: defaultButton},
		},
		Compose: compose,
		Parse:   func(d doc.Document, e widget.Env) widget.Widget { return parse(d, render, e) },
		Events:  events,
		Data:    data,
		Act:     act,
	}
}

// compose writes a new form of one question: its heading, then its
// options ("- ( )" for one choice, "- [ ]" for many) or a line to fill in,
// then the buttons.
func compose(title string, opts map[string]string) ([]byte, error) {
	var b strings.Builder
	if q := opts["question"]; q != "" {
		b.WriteString("## " + q + "\n")
	}
	switch opts["answer"] {
	case "text":
		b.WriteString(">\n")
	default:
		box := "- ( ) "
		if opts["answer"] == "many" {
			box = "- [ ] "
		}
		choices := widget.Lines(opts["choices"])
		if len(choices) == 0 {
			return nil, fmt.Errorf("%w: a question answered by choosing needs choices", widget.ErrBadOption)
		}
		for _, c := range choices {
			b.WriteString(box + c + "\n")
		}
	}
	var buttons []string
	for _, label := range widget.Lines(opts["buttons"]) {
		if strings.ContainsAny(label, "[]") {
			return nil, fmt.Errorf("%w: buttons of a form: %q may not hold [ or ]", widget.ErrBadOption, label)
		}
		buttons = append(buttons, "[ "+label+" ]")
	}
	if len(buttons) > 0 {
		b.WriteString("\n" + strings.Join(buttons, " ") + "\n")
	}
	return widget.NewFile("form", title, b.String()), nil
}
