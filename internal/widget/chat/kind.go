// Package chat is the conversation note: messages with who wrote them and
// when, under a heading per day, written by the user on the board and by
// agents with stickypane say. It is a log with authors, not a chat client:
// no threads, no reactions.
package chat

import (
	"github.com/LeeSwallow/stickypane/internal/doc"
	"github.com/LeeSwallow/stickypane/internal/widget"
)

// Kind registers the chat.
var Kind = widget.Kind{
	Name:     "chat",
	Label:    "Chat",
	Icon:     "✉",
	Hint:     []string{"n", "say", "j k", "scroll"},
	Blurb:    "A conversation in a note: who said what and when, the newest in sight.",
	Example:  "## 2026-10-03\n@min 14:02 can you rerun the tests?\n@claude 14:03 41/41 pass\n",
	Command:  `stickypane say chat "41/41 pass" --as claude`,
	Usage:    "Messages are \"@name HH:MM text\" lines under a \"## YYYY-MM-DD\" heading; indented lines continue one. n on the board writes as you (git's user.name or $USER); stickypane say writes as --as. Each message is a message.added event, so an agent waits for the user with stickypane watch --once --note chat --type message.",
	Keys:     "n enter",
	Tail:     true,
	Template: func(title string) []byte { return widget.NewFile("chat", title, "") },
	Parse:    func(d doc.Document) widget.Widget { return parse(d) },
	Events:   events,
}
