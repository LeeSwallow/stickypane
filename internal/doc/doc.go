// Package doc splits a note file into front matter and body, and edits front
// matter one line at a time so that every other byte of the file is preserved.
package doc

import (
	"errors"
	"strconv"
	"strings"
)

// ErrConflict reports that an Op could not find its target in the document,
// usually because the file changed after the screen last read it.
var ErrConflict = errors.New("note changed on disk")

// Op is an edit expressed as an intent ("move this card to that column") and
// applied to a freshly read document.
type Op interface {
	Apply(d Document) (Document, error)
}

// SetKey sets one front matter key.
type SetKey struct{ Key, Value string }

// Apply implements Op.
func (o SetKey) Apply(d Document) (Document, error) { return d.Set(o.Key, o.Value), nil }

// Document is a note file split into front matter lines and body.
type Document struct {
	// HasFront reports whether the file starts with a closed "---" block.
	HasFront bool
	// Front holds the lines between the fences, without their "\n".
	Front []string
	// Body is everything after the closing fence, byte for byte. Without
	// front matter it is the whole file.
	Body string

	open, close string // the fence lines as written, without "\n"
	closeNL     bool   // the closing fence ended with "\n"
}

// Parse splits src. A front matter block that is never closed is not front
// matter: the whole file becomes the body, so nothing is hidden.
func Parse(src []byte) Document {
	s := string(src)
	lines := strings.SplitAfter(s, "\n")
	if len(lines) < 2 || !isFence(lines[0]) {
		return Document{Body: s}
	}
	for i := 1; i < len(lines); i++ {
		if !isFence(lines[i]) {
			continue
		}
		d := Document{
			HasFront: true,
			open:     strings.TrimSuffix(lines[0], "\n"),
			close:    strings.TrimSuffix(lines[i], "\n"),
			closeNL:  strings.HasSuffix(lines[i], "\n"),
			Body:     strings.Join(lines[i+1:], ""),
		}
		for _, l := range lines[1:i] {
			d.Front = append(d.Front, strings.TrimSuffix(l, "\n"))
		}
		return d
	}
	return Document{Body: s}
}

func isFence(line string) bool { return strings.TrimRight(line, "\r\n") == "---" }

// Bytes returns the file content.
func (d Document) Bytes() []byte {
	if !d.HasFront {
		return []byte(d.Body)
	}
	var b strings.Builder
	b.WriteString(d.open)
	b.WriteString("\n")
	for _, l := range d.Front {
		b.WriteString(l)
		b.WriteString("\n")
	}
	b.WriteString(d.close)
	if d.closeNL {
		b.WriteString("\n")
	}
	b.WriteString(d.Body)
	return []byte(b.String())
}

// find returns the index of the unindented "key: value" line for key, or -1.
func (d Document) find(key string) int {
	for i, l := range d.Front {
		if strings.HasPrefix(l, " ") || strings.HasPrefix(l, "\t") {
			continue
		}
		if k, _, ok := strings.Cut(l, ":"); ok && strings.TrimSpace(k) == key {
			return i
		}
	}
	return -1
}

// Get returns the scalar value of a top-level front matter key.
func (d Document) Get(key string) (string, bool) {
	i := d.find(key)
	if i < 0 {
		return "", false
	}
	_, v, _ := strings.Cut(d.Front[i], ":")
	return unquote(strings.TrimSpace(v)), true
}

// Set returns a copy with key set to value. Only that key's line changes.
// A document without front matter gets a new block.
func (d Document) Set(key, value string) Document {
	line := key + ": " + quote(value)
	out := d
	if !d.HasFront {
		out.HasFront, out.open, out.close, out.closeNL = true, "---", "---", true
		out.Front = []string{line}
		return out
	}
	out.Front = append([]string(nil), d.Front...)
	if i := d.find(key); i >= 0 {
		if strings.HasSuffix(d.Front[i], "\r") {
			line += "\r"
		}
		out.Front[i] = line
		return out
	}
	if strings.HasSuffix(d.open, "\r") {
		line += "\r"
	}
	out.Front = append(out.Front, line)
	return out
}

// Unset returns a copy without key. Only that key's line goes; a document
// that does not have the key is returned as it is.
func (d Document) Unset(key string) Document {
	i := d.find(key)
	if i < 0 {
		return d
	}
	out := d
	out.Front = append(append([]string(nil), d.Front[:i]...), d.Front[i+1:]...)
	return out
}

// Type returns the lower-cased "type" key, or "" when it is absent.
func (d Document) Type() string {
	v, _ := d.Get("type")
	return strings.ToLower(v)
}

// Pinned reports whether "pin" is true.
func (d Document) Pinned() bool {
	v, _ := d.Get("pin")
	return strings.EqualFold(v, "true")
}

func quote(v string) string {
	if v == "" || v != strings.TrimSpace(v) ||
		strings.ContainsAny(v, ":#\"'\n") ||
		strings.ContainsAny(v[:1], "[]{}>|*&!%@`-") {
		return strconv.Quote(v)
	}
	return v
}

func unquote(v string) string {
	if len(v) < 2 {
		return v
	}
	switch {
	case v[0] == '"' && v[len(v)-1] == '"':
		if s, err := strconv.Unquote(v); err == nil {
			return s
		}
		return v[1 : len(v)-1]
	case v[0] == '\'' && v[len(v)-1] == '\'':
		return strings.ReplaceAll(v[1:len(v)-1], "''", "'")
	}
	return v
}

// Lines splits a body into lines without their "\n". A body that ends with a
// newline yields a final empty element, so Join(Lines(b)) == b.
func Lines(body string) []string { return strings.Split(body, "\n") }

// Join is the inverse of Lines.
func Join(lines []string) string { return strings.Join(lines, "\n") }

// EOL returns the suffix a new line needs to match the body's line endings:
// "\r" for CRLF bodies, "" otherwise.
func EOL(body string) string {
	if strings.Contains(body, "\r\n") {
		return "\r"
	}
	return ""
}

// IsDetail reports whether line is an indented, non-blank line: in a list,
// a detail of the item above it.
func IsDetail(line string) bool {
	return (strings.HasPrefix(line, " ") || strings.HasPrefix(line, "\t")) && strings.TrimSpace(line) != ""
}

// OneLine keeps text that becomes a line of a file to one line: line breaks
// and runs of spaces become one space.
func OneLine(s string) string {
	return strings.Join(strings.Fields(strings.NewReplacer("\r", " ", "\n", " ").Replace(s)), " ")
}

// AppendLine returns body with line added at its end, in the body's line
// endings. A body whose last line is unfinished gets it finished first.
func AppendLine(body, line string) string {
	eol := EOL(body)
	if body != "" && !strings.HasSuffix(body, "\n") {
		body += eol + "\n"
	}
	return body + line + eol + "\n"
}

// SetLine replaces line i of lines, one of Lines' elements, with text and
// keeps the "\r" of a CRLF line.
func SetLine(lines []string, i int, text string) {
	if strings.HasSuffix(lines[i], "\r") {
		text += "\r"
	}
	lines[i] = text
}
