package board

import (
	"regexp"
	"strings"
	"time"

	"github.com/LeeSwallow/stickypane/internal/doc"
	"github.com/LeeSwallow/stickypane/internal/when"
)

// cardSpan is a card's place in the body: its first line and the detail
// lines below it, as the half-open line range [start, end).
type cardSpan struct {
	text       string
	start, end int
}

// colSpan is a column: its "## " heading line and the cards below it.
type colSpan struct {
	title string
	head  int
	cards []cardSpan
}

func isHeading(line string) bool { return strings.HasPrefix(line, "## ") }

func isBlank(line string) bool { return strings.TrimSpace(line) == "" }

// cardText is what identifies and labels a card: the line without its list
// marker, who has it and when it moved. A line that is not a list item is
// a card too, so nothing written under a column is ever hidden.
func cardText(line string) string {
	text, _, _ := splitCard(lineText(line))
	return text
}

// lineText is a card's line without its list marker.
func lineText(line string) string {
	line = strings.TrimRight(line, "\r")
	if strings.HasPrefix(line, "- ") || strings.HasPrefix(line, "* ") {
		line = line[2:]
	}
	return strings.TrimSpace(line)
}

// A card may say who has it, as Backlog.md writes it, and when it last
// moved, as Obsidian's Kanban writes dates and times:
//
//   - table widget @codex-1 @{2026-10-03} @@{14:02}
//
// stickypane writes the time itself; an agent writes only the @name.
var (
	movedRe = regexp.MustCompile(`\s*@\{(\d{4}-\d{2}-\d{2})\}(?:\s*@@\{(\d{2}:\d{2})\})?`)
	whoRe   = regexp.MustCompile(`(^|\s)@([\p{L}\p{N}][\p{L}\p{N}._/-]*)`)
)

// splitCard takes who has a card and when it moved out of its text.
func splitCard(s string) (text string, who []string, moved time.Time) {
	if m := movedRe.FindStringSubmatch(s); m != nil {
		clock := "00:00"
		if m[2] != "" {
			clock = m[2]
		}
		moved, _ = when.Parse(m[1] + " " + clock)
	}
	s = movedRe.ReplaceAllString(s, "")
	for _, m := range whoRe.FindAllStringSubmatch(s, -1) {
		who = append(who, m[2])
	}
	s = whoRe.ReplaceAllString(s, "$1")
	return strings.Join(strings.Fields(s), " "), who, moved
}

// unmoved is a card's line without when it moved.
func unmoved(line string) string {
	return strings.TrimRight(movedRe.ReplaceAllString(strings.TrimRight(line, "\r"), ""), " ") + line[len(strings.TrimRight(line, "\r")):]
}

// movedAt writes onto a card's line when it moved, in place of any time it
// had.
func movedAt(line string, t time.Time) string {
	cr := line[len(strings.TrimRight(line, "\r")):]
	return strings.TrimRight(unmoved(strings.TrimRight(line, "\r")), " ") + " @{" + t.Format(when.Date) + "} @@{" + t.Format(when.Clock) + "}" + cr
}

// scan finds the columns and cards in body lines. Every non-blank line under
// a column starts a card and owns the indented lines below it, including
// blank lines inside those details (such as in a code block). Lines before
// the first heading belong to no column.
func scan(lines []string) []colSpan {
	var cols []colSpan
	for i := 0; i < len(lines); i++ {
		line := lines[i]
		if isHeading(line) {
			cols = append(cols, colSpan{title: strings.TrimSpace(line[3:]), head: i})
			continue
		}
		if len(cols) == 0 || isBlank(line) {
			continue
		}
		c := cardSpan{text: cardText(line), start: i, end: i + 1}
		for j := c.end; j < len(lines); j++ {
			if doc.IsDetail(lines[j]) {
				c.end = j + 1
			} else if !isBlank(lines[j]) {
				break
			}
		}
		last := &cols[len(cols)-1]
		last.cards = append(last.cards, c)
		i = c.end - 1
	}
	return cols
}
