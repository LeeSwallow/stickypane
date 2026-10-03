package checklist

import (
	"github.com/LeeSwallow/stickypane/internal/doc"
	"github.com/LeeSwallow/stickypane/internal/widget"
)

// Kind registers the checklist.
var Kind = widget.Kind{
	Name:     "checklist",
	Label:    "Checklist",
	Icon:     "☑",
	Hint:     []string{"j k", "move", "space", "tick", "n", "new item"},
	Blurb:    "Steps to tick off, with a progress bar.",
	Command:  "stickypane todo plan add \"write tests\"",
	Usage:    "\"- [ ]\" and \"- [x]\" lines, shown with a progress bar: the open items first, then the done ones, the most recent first, with when. Ticking an item, with space or `stickypane todo plan check tests` (by a part of its text or \"#2\"), writes the time at the end of its line: \"- [x] tests ✅ 2026-10-03 14:02\".",
	Example:  "- [x] Add endpoint\n- [x] Validate input\n- [ ] Write tests\n",
	Keys:     "j k up down space n",
	Size:     func(doc.Document) string { return widget.SizeHalf },
	Template: func(title string) []byte { return widget.NewFile("checklist", title, "") },
	Parse:    func(d doc.Document) widget.Widget { return parse(d) },
	Tend:     tend,
}
