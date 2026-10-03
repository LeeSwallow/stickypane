package markdown

import (
	"strings"
	"unicode"
)

// inline draws what is inside a line: **bold**, *italic*, ~~struck~~,
// `code`, [links](url) and ![images](src), without their marks; a
// backslash keeps the next mark as written. Unclosed marks stay as text.
func (r renderer) inline(s string) string {
	var b strings.Builder
	rs := []rune(s)
	for i := 0; i < len(rs); i++ {
		c := rs[i]
		switch {
		case c == '\\' && i+1 < len(rs) && unicode.IsPunct(rs[i+1]) || c == '\\' && i+1 < len(rs) && unicode.IsSymbol(rs[i+1]):
			b.WriteRune(rs[i+1])
			i++
		case c == '`':
			n := run(rs, i, '`')
			if end := find(rs, i+n, strings.Repeat("`", n)); end >= 0 {
				b.WriteString(r.s.Code.Render(strings.TrimSpace(string(rs[i+n : end]))))
				i = end + n - 1
				continue
			}
			b.WriteString(string(rs[i : i+n]))
			i += n - 1
		case (c == '*' || c == '_') && i+1 < len(rs) && rs[i+1] == c:
			mark := string([]rune{c, c})
			if end := find(rs, i+2, mark); end > i+2 && (c == '*' || boundary(rs, i)) {
				b.WriteString(r.s.Strong.Render(r.inline(string(rs[i+2 : end]))))
				i = end + 1
				continue
			}
			b.WriteString(mark)
			i++
		case c == '*' || (c == '_' && boundary(rs, i)):
			if end := find(rs, i+1, string(c)); end > i+1 && !unicode.IsSpace(rs[i+1]) {
				b.WriteString(r.s.Emph.Render(r.inline(string(rs[i+1 : end]))))
				i = end
				continue
			}
			b.WriteRune(c)
		case c == '~' && i+1 < len(rs) && rs[i+1] == '~':
			if end := find(rs, i+2, "~~"); end > i+2 {
				b.WriteString(r.s.Muted.Strikethrough(true).Render(r.inline(string(rs[i+2 : end]))))
				i = end + 1
				continue
			}
			b.WriteString("~~")
			i++
		case c == '!' && i+1 < len(rs) && rs[i+1] == '[':
			if text, end, ok := link(rs, i+1); ok {
				b.WriteString(r.s.Muted.Render("[" + text + "]"))
				i = end
				continue
			}
			b.WriteRune(c)
		case c == '[' && i+1 < len(rs) && rs[i+1] == '[':
			// [[note]], [[note#item]] or [[note|alias]]: a link to another
			// note, drawn as its alias or its name.
			if end := find(rs, i+2, "]]"); end > i+2 {
				target := string(rs[i+2 : end])
				if _, alias, ok := strings.Cut(target, "|"); ok {
					target = alias
				}
				b.WriteString(r.s.Link.Render(strings.TrimSpace(target)))
				i = end + 1
				continue
			}
			b.WriteRune(c)
		case c == '[':
			if text, end, ok := link(rs, i); ok {
				b.WriteString(r.s.Link.Render(r.inline(text)))
				i = end
				continue
			}
			b.WriteRune(c)
		case c == '<':
			if end := find(rs, i+1, ">"); end > i+1 && strings.Contains(string(rs[i+1:end]), "://") {
				b.WriteString(r.s.Link.Render(string(rs[i+1 : end])))
				i = end
				continue
			}
			b.WriteRune(c)
		default:
			b.WriteRune(c)
		}
	}
	return b.String()
}

// run is how many times c repeats from i.
func run(rs []rune, i int, c rune) int {
	n := 0
	for i+n < len(rs) && rs[i+n] == c {
		n++
	}
	return n
}

// find returns where mark next starts at or after i, or -1.
func find(rs []rune, i int, mark string) int {
	if i > len(rs) {
		return -1
	}
	if j := strings.Index(string(rs[i:]), mark); j >= 0 {
		return i + len([]rune(string(rs[i:])[:j]))
	}
	return -1
}

// boundary reports whether an underscore at i starts a word, so that
// snake_case is not taken for italics.
func boundary(rs []rune, i int) bool {
	return i == 0 || !(unicode.IsLetter(rs[i-1]) || unicode.IsDigit(rs[i-1]))
}

// link reads "[text](url)" starting at the "[" at i: the text, and where
// the closing ")" is.
func link(rs []rune, i int) (text string, end int, ok bool) {
	close := find(rs, i+1, "](")
	if close < 0 {
		return "", 0, false
	}
	paren := find(rs, close+2, ")")
	if paren < 0 {
		return "", 0, false
	}
	return string(rs[i+1 : close]), paren, true
}
