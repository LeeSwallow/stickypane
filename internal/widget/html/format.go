package html

import (
	stdhtml "html"
	"strings"
)

// blocks are the tags that end a line of text, so that two paragraphs do not
// run together once the tags are gone.
var blocks = map[string]bool{
	"p": true, "div": true, "br": true, "hr": true, "li": true, "tr": true,
	"h1": true, "h2": true, "h3": true, "h4": true, "h5": true, "h6": true,
	"section": true, "article": true, "header": true, "footer": true,
	"ul": true, "ol": true, "table": true, "pre": true, "blockquote": true,
	"title": true,
}

// text is what a page says: no tags, no comments, nothing inside script or
// style, entities decoded, and no more than one blank line in a row.
func text(src string) string {
	var b strings.Builder
	for i := 0; i < len(src); {
		if src[i] != '<' {
			j := strings.IndexByte(src[i:], '<')
			if j < 0 {
				j = len(src) - i
			}
			b.WriteString(src[i : i+j])
			i += j
			continue
		}
		rest := src[i:]
		if strings.HasPrefix(rest, "<!--") {
			i += skipPast(rest, "-->")
			continue
		}
		end := strings.IndexByte(rest, '>')
		if end < 0 { // a '<' with no tag after it is just text
			b.WriteString(rest)
			break
		}
		name := tagName(rest[1:end])
		i += end + 1
		if name == "script" || name == "style" {
			i += skipPast(strings.ToLower(src[i:]), "</"+name)
			if k := strings.IndexByte(src[i:], '>'); k >= 0 {
				i += k + 1
			} else {
				i = len(src)
			}
			continue
		}
		if s := b.String(); blocks[name] && !strings.HasSuffix(s, "\n") {
			b.WriteByte('\n')
		}
	}
	var out []string
	for _, l := range strings.Split(stdhtml.UnescapeString(b.String()), "\n") {
		l = strings.TrimSpace(l)
		if l == "" && (len(out) == 0 || out[len(out)-1] == "") {
			continue
		}
		out = append(out, l)
	}
	for len(out) > 0 && out[len(out)-1] == "" {
		out = out[:len(out)-1]
	}
	return strings.Join(out, "\n")
}

// skipPast is how many bytes of s to skip to get past the first end in it,
// or all of s when it has none.
func skipPast(s, end string) int {
	if i := strings.Index(s, end); i >= 0 {
		return i + len(end)
	}
	return len(s)
}

// tagName is the lower-case name in the text between '<' and '>', without
// the slash of a closing tag.
func tagName(tag string) string {
	tag = strings.TrimPrefix(tag, "/")
	n := strings.IndexAny(tag, " \t\r\n/")
	if n >= 0 {
		tag = tag[:n]
	}
	return strings.ToLower(tag)
}
