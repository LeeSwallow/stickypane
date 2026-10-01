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
	barWidth   = 20 // widest the progress bar gets
	formatHint = `no items yet: add "- [ ] task" lines`
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
	Icon:     "☑",
	Hint:     []string{"j k", "move", "space", "tick", "n", "new item"},
	Blurb:    "Steps to tick off, with a progress bar.",
	Example:  "- [x] Add endpoint\n- [x] Validate input\n- [ ] Write tests\n",
	Keys:     "j k up down space n",
	Size:     func(doc.Document) string { return widget.SizeHalf },
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
	return widget.Good.Render(strings.Repeat("▓", fill)) + widget.Faint.Render(strings.Repeat("░", w-fill)) + label
}

// Summary implements widget.Widget: done over total.
func (c *Checklist) Summary() string {
	done, total := c.counts()
	if total == 0 {
		return ""
	}
	return fmt.Sprintf("%d/%d", done, total)
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

// Draw implements widget.Widget. Every line of the note is shown: items with
// their checkbox, everything else as it is. Long items wrap.
func (c *Checklist) Draw(width int, active bool) (string, widget.Span) {
	done, total := c.counts()
	if total == 0 {
		lines := []string{widget.Faint.Render(widget.Truncate(formatHint, width))}
		for _, e := range c.entries {
			if e.text != "" {
				lines = append(lines, widget.Wrap(widget.Clean(e.text), max(width, 1))...)
			}
		}
		return widget.Fit(strings.Join(lines, "\n"), width), widget.NoSpan
	}
	c.clamp()
	lines := []string{bar(done, total, width), ""}
	at := widget.NoSpan
	for i, e := range c.entries {
		if !e.item {
			for _, l := range widget.Wrap(widget.Clean(e.text), max(width, 1)) {
				if strings.HasPrefix(e.text, "#") {
					l = widget.Bold.Render(l)
				}
				lines = append(lines, l)
			}
			continue
		}
		mark, done := "☐ ", func(s string) string { return s }
		if e.checked {
			// A finished item steps back so the open ones stand out.
			mark, done = "☑ ", func(s string) string { return widget.Struck.Render(s) }
		}
		selected := active && c.items[c.cursor] == i
		for j, l := range widget.Wrap(widget.Clean(e.text), max(width-4, 1)) {
			switch {
			case j == 0 && selected:
				at.Start = len(lines)
				lines = append(lines, widget.Selected.Render("› "+mark+l))
			case j == 0:
				lines = append(lines, "  "+mark+done(l))
			case selected:
				lines = append(lines, widget.Selected.Render("    "+l))
			default:
				lines = append(lines, "    "+done(l))
			}
		}
		if selected {
			at.End = len(lines) // every wrapped line of the item
		}
	}
	return widget.Fit(strings.Join(lines, "\n"), width), at
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
	case "space":
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
	nc.cursor = c.cursor
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
