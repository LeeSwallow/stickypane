package chat

import (
	"fmt"
	"strings"

	"github.com/LeeSwallow/stickypane/internal/doc"
	"github.com/LeeSwallow/stickypane/internal/widget"
)

// message is one thing said.
type message struct {
	day, who, clock, text string
}

// read reads the messages in order. Indented lines continue the message
// above; other lines are kept as messages of no one, so nothing is hidden.
func read(d doc.Document) []message {
	var out []message
	day := ""
	for _, l := range doc.Lines(d.Body) {
		l = strings.TrimRight(l, "\r")
		switch {
		case strings.TrimSpace(l) == "":
		case dayRe.MatchString(l):
			day = dayRe.FindStringSubmatch(l)[1]
		case messageRe.MatchString(l):
			m := messageRe.FindStringSubmatch(l)
			out = append(out, message{day: day, who: m[1], clock: m[2], text: m[3]})
		case doc.IsDetail(l) && len(out) > 0:
			out[len(out)-1].text += "\n" + strings.TrimSpace(l)
		default:
			out = append(out, message{day: day, text: strings.TrimSpace(l)})
		}
	}
	return out
}

func parse(d doc.Document) *Chat { return &Chat{messages: read(d)} }

// Summary implements widget.Widget: how many messages.
func (c *Chat) Summary() string {
	switch n := len(c.messages); n {
	case 0:
		return ""
	case 1:
		return widget.T("1 message")
	default:
		return fmt.Sprintf(widget.T("%d messages"), n)
	}
}
