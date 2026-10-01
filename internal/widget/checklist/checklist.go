// Package checklist is the progress note: "- [ ]" and "- [x]" lines.
package checklist

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/LeeSwallow/stickypane/internal/doc"
	"github.com/LeeSwallow/stickypane/internal/widget"
)

const (
	previewOpen = 4  // open items shown on the board
	barWidth    = 20 // widest the progress bar gets
	formatHint  = `no items yet: add "- [ ] task" lines`
)

// itemRe splits a checkbox line into: prefix up to "[", the mark, "] ", text.
var itemRe = regexp.MustCompile(`^(\s*[-*] \[)([ xX])(\] ?)(.*)$`)

// isDetail reports whether line is an indented, non-blank line: a detail of
// the item above it.
func isDetail(line string) bool {
	return (strings.HasPrefix(line, " ") || strings.HasPrefix(line, "\t")) && strings.TrimSpace(line) != ""
}

// Kind registers the checklist.
var Kind = widget.Kind{
	Name:     "checklist",
	Label:    "Checklist",
	Template: func(title string) []byte { return widget.NewFile("checklist", title, "") },
	Parse:    func(d doc.Document) widget.Widget { return parse(d) },
}

type entry struct {
	text    string
	item    bool
	checked bool
}

// Checklist is the widget for a checklist note.
type Checklist struct {
	entries []entry
	items   []int // indexes of the checkbox entries
	cursor  int   // index into items
	offset  int
}

func parse(d doc.Document) *Checklist {
	c := &Checklist{}
	for _, line := range doc.Lines(d.Body) {
		line = strings.TrimRight(line, "\r")
		if m := itemRe.FindStringSubmatch(line); m != nil {
			c.items = append(c.items, len(c.entries))
			c.entries = append(c.entries, entry{text: strings.TrimSpace(m[4]), item: true, checked: m[2] != " "})
			continue
		}
		c.entries = append(c.entries, entry{text: strings.TrimRight(line, " \t")})
	}
	for n := len(c.entries); n > 0 && !c.entries[n-1].item && c.entries[n-1].text == ""; n = len(c.entries) {
		c.entries = c.entries[:n-1]
	}
	return c
}

func (c *Checklist) counts() (done, total int) {
	for _, i := range c.items {
		if c.entries[i].checked {
			done++
		}
	}
	return done, len(c.items)
}

func bar(done, total, width int) string {
	label := fmt.Sprintf(" %d/%d", done, total)
	w := min(barWidth, width-widget.Width(label))
	if w < 1 {
		return widget.Truncate(strings.TrimSpace(label), width)
	}
	fill := done * w / total
	return strings.Repeat("▓", fill) + strings.Repeat("░", w-fill) + label
}

// unshaped draws a checklist that has no items: the format hint followed by
// the text that is there, so a note that does not fit the shape stays visible.
func (c *Checklist) unshaped(width, height int) string {
	lines := []string{widget.Faint.Render(widget.Truncate(formatHint, width))}
	for _, e := range c.entries {
		if e.text != "" {
			lines = append(lines, widget.Truncate(widget.Clean(e.text), width))
		}
	}
	return strings.Join(widget.Window(lines, 0, height), "\n")
}

// nth is how many items above the cursor share the selected item's text. It
// tells an Op which of several identical items is meant.
func (c *Checklist) nth() int {
	text := c.entries[c.items[c.cursor]].text
	n := 0
	for _, i := range c.items[:c.cursor] {
		if c.entries[i].text == text {
			n++
		}
	}
	return n
}

// Preview implements widget.Widget.
func (c *Checklist) Preview(width int) string {
	done, total := c.counts()
	if total == 0 {
		return c.unshaped(width, previewOpen+1)
	}
	lines := []string{bar(done, total, width)}
	open := 0
	for _, i := range c.items {
		if e := c.entries[i]; !e.checked {
			if open++; open <= previewOpen {
				lines = append(lines, widget.Truncate("☐ "+widget.Clean(e.text), width))
			}
		}
	}
	switch {
	case open == 0:
		lines = append(lines, "✓ all done")
	case open > previewOpen:
		lines = append(lines, widget.Faint.Render(fmt.Sprintf("+%d more", open-previewOpen)))
	}
	return strings.Join(lines, "\n")
}

// View implements widget.Widget.
func (c *Checklist) View(width, height int) string {
	done, total := c.counts()
	if total == 0 {
		return c.unshaped(width, height)
	}
	c.clamp()
	lines := []string{bar(done, total, width), ""}
	cursorLine := 0
	for i, e := range c.entries {
		if !e.item {
			lines = append(lines, widget.Truncate(widget.Clean(e.text), width))
			continue
		}
		mark := "☐ "
		if e.checked {
			mark = "☑ "
		}
		text := widget.Truncate(mark+widget.Clean(e.text), width-2)
		if c.items[c.cursor] == i {
			cursorLine = len(lines)
			lines = append(lines, widget.Selected.Render("› "+text))
		} else {
			lines = append(lines, "  "+text)
		}
	}
	if cursorLine < c.offset {
		c.offset = cursorLine
	}
	if cursorLine >= c.offset+height {
		c.offset = cursorLine - height + 1
	}
	return strings.Join(widget.Window(lines, c.offset, height), "\n")
}

func (c *Checklist) clamp() {
	c.cursor = max(min(c.cursor, len(c.items)-1), 0)
}

// Update implements widget.Widget.
func (c *Checklist) Update(key string) (widget.Widget, widget.Result) {
	var res widget.Result
	switch key {
	case "j", "down":
		c.cursor++
	case "k", "up":
		c.cursor--
	case "space", "x":
		c.clamp()
		if len(c.items) > 0 {
			e := &c.entries[c.items[c.cursor]]
			res.Op = Toggle{Text: e.text, Checked: !e.checked, Nth: c.nth()}
			e.checked = !e.checked
		}
	case "n":
		res.Prompt = &widget.Prompt{
			Label:  "New item",
			Submit: func(text string) doc.Op { return AddItem{Text: text} },
		}
	}
	c.clamp()
	return c, res
}

// Sync implements widget.Widget.
func (c *Checklist) Sync(d doc.Document) widget.Widget {
	nc := parse(d)
	nc.cursor, nc.offset = c.cursor, c.offset
	nc.clamp()
	return nc
}

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
			for at < len(lines) && isDetail(lines[at]) {
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
