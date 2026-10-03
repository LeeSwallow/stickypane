// Package markdown draws Markdown for the terminal: the part of it notes
// use, in the colors it is given, never wider than the width. It replaces
// a general renderer (and its HTML, emoji and syntax-highlighting
// libraries) with what the board needs.
//
// Headings, paragraphs, lists (bullets, numbers, task boxes, nested),
// quotes, code blocks, rules and tables are drawn; bold, italic, struck,
// code, links and images inside a line are drawn without their marks.
// Anything else is shown as written, so nothing is lost.
package markdown

import (
	"regexp"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

// Styles are the colors a rendering uses. A zero style draws plain text.
type Styles struct {
	Heading lipgloss.Style // headings
	Minor   lipgloss.Style // the smallest heading
	Muted   lipgloss.Style // quotes, rules, image names
	Link    lipgloss.Style // link text
	Code    lipgloss.Style // code, inline and in blocks
	Mark    lipgloss.Style // bullets, numbers and task boxes
	Strong  lipgloss.Style // **bold**
	Emph    lipgloss.Style // *italic*
}

var (
	headingRe = regexp.MustCompile(`^ {0,3}(#{1,6})\s+(.*?)\s*#*\s*$`)
	ruleRe    = regexp.MustCompile(`^ {0,3}([-*_])( *[-*_]){2,} *$`)
	itemRe    = regexp.MustCompile(`^(\s*)([-*+]|\d{1,9}[.)])\s+(.*)$`)
	taskRe    = regexp.MustCompile(`^\[([ xX])\]\s+(.*)$`)
	fenceRe   = regexp.MustCompile("^ {0,3}(```+|~~~+)")
	setext1Re = regexp.MustCompile(`^ {0,3}=+\s*$`)
	setext2Re = regexp.MustCompile(`^ {0,3}-+\s*$`)
)

// Render draws src at width with the styles s.
func Render(src string, width int, s Styles) string {
	width = max(width, 1)
	r := renderer{s: s, width: width}
	lines := strings.Split(strings.ReplaceAll(src, "\r\n", "\n"), "\n")
	var blocks []string
	for i := 0; i < len(lines); {
		line := lines[i]
		var block string
		switch {
		case strings.TrimSpace(line) == "":
			i++
			continue
		case fenceRe.MatchString(line):
			block, i = r.fence(lines, i)
		case headingRe.MatchString(line):
			m := headingRe.FindStringSubmatch(line)
			block, i = r.heading(len(m[1]), m[2]), i+1
		case ruleRe.MatchString(line):
			block, i = s.Muted.Render(strings.Repeat("─", width)), i+1
		case strings.HasPrefix(strings.TrimLeft(line, " "), ">"):
			block, i = r.quote(lines, i)
		case isTable(lines, i):
			block, i = r.table(lines, i)
		case itemRe.MatchString(line):
			block, i = r.list(lines, i)
		case strings.HasPrefix(line, "    ") || strings.HasPrefix(line, "\t"):
			block, i = r.indented(lines, i)
		default:
			block, i = r.paragraph(lines, i)
		}
		blocks = append(blocks, block)
	}
	// The last safeguard: a width too small for a wide character, or for a
	// word the wrapping could not break, cuts the line rather than widen it.
	out := strings.Split(strings.Join(blocks, "\n\n"), "\n")
	for i, l := range out {
		if ansi.StringWidth(l) > width {
			out[i] = ansi.Truncate(l, width, "")
		}
	}
	return strings.Join(out, "\n")
}

type renderer struct {
	s     Styles
	width int
}

// wrap breaks text at the width, words first, and cuts a word longer than
// a line, so that no line is wider.
func wrap(text string, width int) []string {
	return strings.Split(ansi.Wrap(text, max(width, 1), ""), "\n")
}

func (r renderer) heading(level int, text string) string {
	style := r.s.Heading
	if level == 6 {
		style = r.s.Minor
	}
	var out []string
	for _, l := range wrap(r.inline(text), r.width) {
		out = append(out, style.Render(l))
	}
	return strings.Join(out, "\n")
}

// paragraph joins lines up to a blank line or another block, keeping hard
// breaks (two spaces or a backslash at the end of a line). A single line
// underlined with === or --- is a heading.
func (r renderer) paragraph(lines []string, i int) (string, int) {
	if i+1 < len(lines) {
		switch {
		case setext1Re.MatchString(lines[i+1]):
			return r.heading(1, strings.TrimSpace(lines[i])), i + 2
		case setext2Re.MatchString(lines[i+1]) && strings.TrimSpace(lines[i+1]) != "":
			return r.heading(2, strings.TrimSpace(lines[i])), i + 2
		}
	}
	var text strings.Builder
	for ; i < len(lines) && !startsBlock(lines, i); i++ {
		l := lines[i]
		hard := strings.HasSuffix(l, "  ") || strings.HasSuffix(l, "\\")
		text.WriteString(strings.TrimSpace(strings.TrimSuffix(l, "\\")))
		if hard {
			text.WriteString("\n")
		} else {
			text.WriteString(" ")
		}
	}
	var out []string
	for _, part := range strings.Split(strings.TrimSpace(text.String()), "\n") {
		out = append(out, wrap(r.inline(strings.TrimSpace(part)), r.width)...)
	}
	return strings.Join(out, "\n"), i
}

// startsBlock reports whether line i ends a paragraph: a blank line or the
// start of another block.
func startsBlock(lines []string, i int) bool {
	l := lines[i]
	return strings.TrimSpace(l) == "" || fenceRe.MatchString(l) || headingRe.MatchString(l) ||
		ruleRe.MatchString(l) || strings.HasPrefix(strings.TrimLeft(l, " "), ">") ||
		itemRe.MatchString(l) || isTable(lines, i)
}

// fence draws a fenced code block as written, two cells in, each line cut
// to the width.
func (r renderer) fence(lines []string, i int) (string, int) {
	mark := fenceRe.FindStringSubmatch(lines[i])[1]
	var body []string
	for i++; i < len(lines); i++ {
		if strings.HasPrefix(strings.TrimLeft(lines[i], " "), mark) {
			i++
			break
		}
		body = append(body, lines[i])
	}
	return r.code(body), i
}

// indented draws a code block written with four spaces.
func (r renderer) indented(lines []string, i int) (string, int) {
	var body []string
	for ; i < len(lines) && (strings.HasPrefix(lines[i], "    ") || strings.HasPrefix(lines[i], "\t") || strings.TrimSpace(lines[i]) == ""); i++ {
		body = append(body, strings.TrimPrefix(strings.TrimPrefix(lines[i], "    "), "\t"))
	}
	for len(body) > 0 && strings.TrimSpace(body[len(body)-1]) == "" {
		body = body[:len(body)-1]
	}
	return r.code(body), i
}

func (r renderer) code(body []string) string {
	out := make([]string, 0, len(body))
	for _, l := range body {
		l = strings.ReplaceAll(l, "\t", "    ")
		out = append(out, ansi.Truncate("  "+r.s.Code.Render(l), r.width, ""))
	}
	return strings.Join(out, "\n")
}

// quote draws a block quote: its text, rendered, under a bar.
func (r renderer) quote(lines []string, i int) (string, int) {
	var inner []string
	for ; i < len(lines) && strings.HasPrefix(strings.TrimLeft(lines[i], " "), ">"); i++ {
		l := strings.TrimPrefix(strings.TrimLeft(lines[i], " "), ">")
		inner = append(inner, strings.TrimPrefix(l, " "))
	}
	bar := r.s.Muted.Render("│ ")
	sub := renderer{s: r.s, width: max(r.width-2, 1)}
	body := Render(strings.Join(inner, "\n"), sub.width, sub.s)
	var out []string
	for _, l := range strings.Split(body, "\n") {
		out = append(out, bar+r.s.Muted.Render(ansi.Strip(l)))
	}
	return strings.Join(out, "\n"), i
}

// list draws consecutive list items: bullets, numbers or task boxes,
// nested by indentation. Lines indented under an item continue it, so an
// item written over two lines is one item, and its wrapped lines hang
// under its text.
func (r renderer) list(lines []string, i int) (string, int) {
	type item struct {
		indent int
		marker string
		text   string
	}
	var items []item
	for i < len(lines) {
		l := lines[i]
		if m := itemRe.FindStringSubmatch(l); m != nil {
			items = append(items, item{indent: len(strings.ReplaceAll(m[1], "\t", "    ")), marker: m[2], text: m[3]})
			i++
			continue
		}
		if strings.TrimSpace(l) == "" {
			// A blank line ends the list unless an indented line follows.
			if i+1 < len(lines) && len(items) > 0 && strings.HasPrefix(lines[i+1], "  ") && !itemRe.MatchString(lines[i+1]) {
				i++
				continue
			}
			break
		}
		if len(items) == 0 || (!strings.HasPrefix(l, " ") && !strings.HasPrefix(l, "\t")) {
			break
		}
		items[len(items)-1].text += " " + strings.TrimSpace(l)
		i++
	}
	// Depth by indentation: each deeper indent than the one above nests.
	var stack []int
	var out []string
	for _, it := range items {
		for len(stack) > 0 && it.indent < stack[len(stack)-1] {
			stack = stack[:len(stack)-1]
		}
		if len(stack) == 0 || it.indent > stack[len(stack)-1] {
			stack = append(stack, it.indent)
		}
		depth := len(stack) - 1
		mark := "•"
		if it.marker[0] >= '0' && it.marker[0] <= '9' {
			mark = strings.TrimRight(it.marker, ".)") + "."
		}
		text := it.text
		if m := taskRe.FindStringSubmatch(text); m != nil {
			mark, text = "☐", m[2]
			if m[1] != " " {
				mark = "☑"
			}
		}
		lead := strings.Repeat("  ", depth)
		hang := lead + strings.Repeat(" ", ansi.StringWidth(mark)+1)
		room := r.width - ansi.StringWidth(hang)
		if room < 4 { // too narrow to hang: wrap at the full width
			for _, l := range wrap(lead+r.s.Mark.Render(mark)+" "+r.inline(text), r.width) {
				out = append(out, l)
			}
			continue
		}
		for j, l := range wrap(r.inline(text), room) {
			if j == 0 {
				out = append(out, lead+r.s.Mark.Render(mark)+" "+l)
			} else {
				out = append(out, hang+l)
			}
		}
	}
	return strings.Join(out, "\n"), i
}
