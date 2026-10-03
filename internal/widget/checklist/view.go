// Package checklist is the progress note: "- [ ]" and "- [x]" lines.
package checklist

import (
	"fmt"
	"strings"

	"github.com/LeeSwallow/stickypane/internal/doc"
	"github.com/LeeSwallow/stickypane/internal/widget"
)

const (
	barWidth   = 20 // widest the progress bar gets
	formatHint = `no items yet: add "- [ ] task" lines`
)

// Checklist is the widget for a checklist note: the list and where the
// cursor is.
type Checklist struct {
	list
	cursor int           // index into items
	drawn  []widget.Span // the lines of each item in the last Draw
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
		lines := []string{widget.Faint.Render(widget.Truncate(widget.T(formatHint), width))}
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
	c.drawn = c.drawn[:0]
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
		from := len(lines)
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
		c.drawn = append(c.drawn, widget.Span{Start: from, End: len(lines)})
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

// Click implements widget.Clicker: a press on an item ticks or unticks it.
func (c *Checklist) Click(line, _ int) (widget.Widget, widget.Result, bool) {
	for i, s := range c.drawn {
		if i < len(c.items) && line >= s.Start && line < s.End {
			c.cursor = i
			w, res := c.Update("space")
			return w, res, true
		}
	}
	return c, widget.Result{}, false
}

// Sync implements widget.Widget.
func (c *Checklist) Sync(d doc.Document) widget.Widget {
	nc := parse(d)
	nc.cursor = c.cursor
	nc.clamp()
	return nc
}
