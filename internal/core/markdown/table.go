package markdown

import (
	"regexp"
	"strings"

	"github.com/charmbracelet/x/ansi"
)

var separatorRe = regexp.MustCompile(`^\s*\|?\s*:?-+:?\s*(\|\s*:?-+:?\s*)*\|?\s*$`)

// isTable reports whether line i starts a table: a row of cells, then a
// line of dashes under it.
func isTable(lines []string, i int) bool {
	return strings.Contains(lines[i], "|") && i+1 < len(lines) && separatorRe.MatchString(lines[i+1]) && strings.Contains(lines[i+1], "-")
}

// cells splits a table row at its bars; a bar after a backslash is text.
func cells(row string) []string {
	row = strings.TrimSpace(row)
	row = strings.TrimPrefix(strings.TrimSuffix(row, "|"), "|")
	var out []string
	var cur strings.Builder
	for i := 0; i < len(row); i++ {
		switch {
		case row[i] == '\\' && i+1 < len(row) && row[i+1] == '|':
			cur.WriteByte('|')
			i++
		case row[i] == '|':
			out = append(out, strings.TrimSpace(cur.String()))
			cur.Reset()
		default:
			cur.WriteByte(row[i])
		}
	}
	return append(out, strings.TrimSpace(cur.String()))
}

// table draws a table as aligned columns two cells apart: the header in
// bold, a rule, then the rows. Columns too wide for the width are narrowed,
// the widest first, and their cells cut.
func (r renderer) table(lines []string, i int) (string, int) {
	head := cells(lines[i])
	var rows [][]string
	for i += 2; i < len(lines) && strings.Contains(lines[i], "|") && strings.TrimSpace(lines[i]) != ""; i++ {
		rows = append(rows, cells(lines[i]))
	}
	n := len(head)
	for _, row := range rows {
		n = max(n, len(row))
	}
	draw := func(row []string) []string {
		out := make([]string, n)
		for c := range out {
			if c < len(row) {
				out[c] = r.inline(row[c])
			}
		}
		return out
	}
	all := append([][]string{draw(head)}, nil)[:1]
	for _, row := range rows {
		all = append(all, draw(row))
	}
	widths := make([]int, n)
	for _, row := range all {
		for c, cell := range row {
			widths[c] = max(widths[c], ansi.StringWidth(cell))
		}
	}
	for total(widths) > r.width {
		widest := 0
		for c := range widths {
			if widths[c] > widths[widest] {
				widest = c
			}
		}
		if widths[widest] <= 1 {
			break
		}
		widths[widest]--
	}
	line := func(row []string, bold bool) string {
		parts := make([]string, n)
		for c, cell := range row {
			cell = ansi.Truncate(cell, widths[c], "…")
			if bold {
				cell = r.s.Strong.Bold(true).Render(cell)
			}
			parts[c] = cell + strings.Repeat(" ", max(widths[c]-ansi.StringWidth(cell), 0))
		}
		return ansi.Truncate(strings.TrimRight(strings.Join(parts, "  "), " "), r.width, "")
	}
	out := []string{line(all[0], true)}
	rule := make([]string, n)
	for c, w := range widths {
		rule[c] = strings.Repeat("─", w)
	}
	out = append(out, ansi.Truncate(r.s.Muted.Render(strings.Join(rule, "  ")), r.width, ""))
	for _, row := range all[1:] {
		out = append(out, line(row, false))
	}
	return strings.Join(out, "\n"), i
}

// total is the cells a table takes: its columns and the gaps between them.
func total(widths []int) int {
	t := 2 * max(len(widths)-1, 0)
	for _, w := range widths {
		t += w
	}
	return t
}
