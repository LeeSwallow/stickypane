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

// Kind registers the checklist.
var Kind = widget.Kind{
	Name:     "checklist",
	Label:    "Checklist",
	Template: func(title string) []byte { return widget.NewFile("checklist", title, "- [ ] \n") },
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

// Preview implements widget.Widget.
func (c *Checklist) Preview(width int) string {
	done, total := c.counts()
	if total == 0 {
		return widget.Faint.Render(widget.Truncate(formatHint, width))
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
		return widget.Faint.Render(widget.Truncate(formatHint, width))
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
			res.Op = Toggle{Text: e.text, Checked: !e.checked}
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

// Toggle sets the first item with Text that is not yet in state Checked.
type Toggle struct {
	Text    string
	Checked bool
}

// Apply implements doc.Op.
func (o Toggle) Apply(d doc.Document) (doc.Document, error) {
	lines := doc.Lines(d.Body)
	for i, line := range lines {
		raw := strings.TrimSuffix(line, "\r")
		m := itemRe.FindStringSubmatch(raw)
		if m == nil || strings.TrimSpace(m[4]) != o.Text || (m[2] != " ") == o.Checked {
			continue
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

// AddItem appends an unchecked item after the last checkbox line, or at the
// end of the body when there is none.
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
		}
	}
	out := make([]string, 0, len(lines)+1)
	out = append(out, lines[:at]...)
	out = append(out, "- [ ] "+o.Text+doc.EOL(d.Body))
	d.Body = doc.Join(append(out, lines[at:]...))
	return d, nil
}
