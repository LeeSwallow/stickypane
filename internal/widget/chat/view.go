package chat

import (
	"strings"

	"github.com/LeeSwallow/stickypane/internal/doc"
	"github.com/LeeSwallow/stickypane/internal/widget"
)

// Chat is the widget for a chat note. It has no cursor: the pane follows
// its end, as a log does.
type Chat struct {
	messages []message
	env      widget.Env
}

// Draw implements widget.Widget: a faint line per day, then each message
// with who said it, in their color, and when; its text hangs under it.
func (c *Chat) Draw(width int, _ bool) (string, widget.Span) {
	if len(c.messages) == 0 {
		return c.env.Styles.Faint.Render(widget.Truncate(c.env.T("nothing said yet: n to say something"), width)), widget.NoSpan
	}
	var lines []string
	day := "\x00"
	for _, m := range c.messages {
		if m.day != day {
			day = m.day
			if day != "" {
				lines = append(lines, c.env.Styles.Faint.Render(widget.Truncate("── "+day+" ", width)))
			}
		}
		head := ""
		if m.who != "" {
			head = c.env.NameStyle(m.who).Render("@"+widget.Clean(m.who)) + " " + c.env.Styles.Faint.Render(m.clock) + " "
		}
		text := widget.Unlink(widget.Clean(m.text))
		first := true
		for _, part := range strings.Split(text, "\n") {
			room := width - 2
			if first {
				room = width - widget.Width(head)
			}
			for _, l := range widget.Wrap(part, max(room, 1)) {
				if first {
					lines = append(lines, head+l)
					first = false
				} else {
					lines = append(lines, "  "+l)
				}
			}
		}
	}
	return widget.Fit(strings.Join(lines, "\n"), width), widget.NoSpan
}

// Update implements widget.Widget: n or enter asks for a message and says
// it as the user.
func (c *Chat) Update(key string) (widget.Widget, widget.Result) {
	if key != "n" && key != "enter" {
		return c, widget.Result{}
	}
	who := c.env.Who()
	return c, widget.Result{Prompt: &widget.Prompt{
		Label: "Say",
		Submit: func(text string) doc.Op {
			return Say{Who: who, Text: text, At: c.env.Now()}
		},
	}}
}

// Sync implements widget.Widget.
func (c *Chat) Sync(d doc.Document) widget.Widget { return parse(d, c.env) }
