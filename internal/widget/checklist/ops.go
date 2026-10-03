package checklist

import (
	"strings"

	"github.com/LeeSwallow/stickypane/internal/doc"
	"github.com/LeeSwallow/stickypane/internal/widget"
)

// Toggle sets an item to state Checked. The item is the one with Text; when
// several items share that text, Nth counts from zero to say which. An item
// that is already in that state is a conflict.
type Toggle struct {
	Text    string
	Checked bool
	Nth     int
}

// Apply implements doc.Op.
func (o Toggle) Apply(d doc.Document) (doc.Document, error) {
	lines := doc.Lines(d.Body)
	nth := o.Nth
	for i, line := range lines {
		raw := strings.TrimSuffix(line, "\r")
		m := itemRe.FindStringSubmatch(raw)
		if m == nil || strings.TrimSpace(m[4]) != o.Text {
			continue
		}
		if nth > 0 {
			nth--
			continue
		}
		if nth < 0 || (m[2] != " ") == o.Checked {
			break // not the item the screen showed
		}
		mark := " "
		if o.Checked {
			mark = "x"
		}
		lines[i] = m[1] + mark + m[3] + m[4] + line[len(raw):]
		d.Body = doc.Join(lines)
		return d, nil
	}
	return d, doc.ErrConflict
}

// AddItem appends an unchecked item after the last checkbox line and the
// indented lines that belong to it, or at the end of the body when there is
// no checkbox yet.
type AddItem struct{ Text string }

// Apply implements doc.Op.
func (o AddItem) Apply(d doc.Document) (doc.Document, error) {
	lines := doc.Lines(d.Body)
	at := len(lines)
	if lines[at-1] == "" {
		at-- // keep the final newline at the end
	}
	for i, line := range lines {
		if itemRe.MatchString(strings.TrimSuffix(line, "\r")) {
			at = i + 1
			for at < len(lines) && doc.IsDetail(lines[at]) {
				at++
			}
		}
	}
	out := make([]string, 0, len(lines)+1)
	out = append(out, lines[:at]...)
	out = append(out, "- [ ] "+o.Text+doc.EOL(d.Body))
	d.Body = doc.Join(append(out, lines[at:]...))
	return d, nil
}

// Check sets an item to state Checked for a caller that names the item
// instead of pointing at it: Item is the item's text, a part of it, or its
// position ("#2"). An item already in that state is left as it is.
type Check struct {
	Item    string
	Checked bool
}

// Apply implements doc.Op.
func (o Check) Apply(d doc.Document) (doc.Document, error) {
	lines := doc.Lines(d.Body)
	var names []string
	var at []int
	for i, line := range lines {
		if m := itemRe.FindStringSubmatch(strings.TrimSuffix(line, "\r")); m != nil {
			names = append(names, strings.TrimSpace(m[4]))
			at = append(at, i)
		}
	}
	n, err := widget.Pick(names, o.Item, "item")
	if err != nil {
		return d, err
	}
	raw := strings.TrimSuffix(lines[at[n]], "\r")
	m := itemRe.FindStringSubmatch(raw)
	mark := " "
	if o.Checked {
		mark = "x"
	}
	lines[at[n]] = m[1] + mark + m[3] + m[4] + lines[at[n]][len(raw):]
	d.Body = doc.Join(lines)
	return d, nil
}
