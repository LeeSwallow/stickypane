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

func parse(d doc.Document, e widget.Env) *Chat { return &Chat{messages: read(d), env: e} }

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

// said is a message as a screen other than the terminal reads it.
type said struct {
	Day   string `json:"day"`
	Who   string `json:"who"`
	Clock string `json:"clock"`
	Text  string `json:"text"`
}

// data is the messages in order, from the same read the board draws.
func data(d doc.Document, _ widget.Env) any {
	out := struct {
		Messages []said `json:"messages"`
	}{Messages: []said{}}
	for _, m := range read(d) {
		out.Messages = append(out.Messages, said{m.day, m.who, m.clock, m.text})
	}
	return out
}
