package board

import (
	"strings"

	"github.com/LeeSwallow/stickypane/internal/doc"
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
// marker. A line that is not a list item is a card too, so nothing written
// under a column is ever hidden.
func cardText(line string) string {
	if strings.HasPrefix(line, "- ") || strings.HasPrefix(line, "* ") {
		line = line[2:]
	}
	return strings.TrimSpace(line)
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
