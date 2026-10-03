package kinds

import (
	"regexp"
	"strings"
)

// listItem matches the first line of a list item: its indent, its marker
// and the text after it.
var listItem = regexp.MustCompile(`^(\s*)([-*+]|\d{1,9}[.)])\s+(\S.*)$`)

// joinListLines puts the lines that continue a list item back on the item's
// line. Glamour draws such a soft line break as a new line at the left
// edge, which reads as a broken item; in a paragraph it already joins them.
// A line ending in a hard break (two spaces or a backslash), a blank line,
// a new item or block, and anything inside a fenced code block are left as
// they are.
func joinListLines(md string) string {
	lines := strings.Split(md, "\n")
	out := make([]string, 0, len(lines))
	fence := ""
	inItem := false // the previous line is a list item's text
	itemIndent := 0
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if fence != "" {
			if strings.HasPrefix(trimmed, fence) {
				fence = ""
			}
			out = append(out, line)
			continue
		}
		if strings.HasPrefix(trimmed, "```") || strings.HasPrefix(trimmed, "~~~") {
			fence = trimmed[:3]
			inItem = false
			out = append(out, line)
			continue
		}
		if m := listItem.FindStringSubmatch(line); m != nil {
			inItem = true
			itemIndent = len(m[1])
			out = append(out, line)
			continue
		}
		indent := len(line) - len(strings.TrimLeft(line, " \t"))
		prev := ""
		if len(out) > 0 {
			prev = out[len(out)-1]
		}
		hardBreak := strings.HasSuffix(prev, "  ") || strings.HasSuffix(prev, `\`)
		if inItem && trimmed != "" && indent > itemIndent && !hardBreak && !startsBlock(trimmed) {
			out[len(out)-1] = strings.TrimRight(prev, " ") + " " + trimmed
			continue
		}
		inItem = false
		out = append(out, line)
	}
	return strings.Join(out, "\n")
}

// startsBlock reports whether a line opens something of its own: a quote,
// a heading, a table row or an HTML block.
func startsBlock(trimmed string) bool {
	return strings.HasPrefix(trimmed, ">") || strings.HasPrefix(trimmed, "#") ||
		strings.HasPrefix(trimmed, "|") || strings.HasPrefix(trimmed, "<")
}
