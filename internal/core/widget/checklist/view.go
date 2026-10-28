// Package checklist is the progress note: "- [ ]" and "- [x]" lines.
package checklist

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/LeeSwallow/stickypane/internal/core/doc"
	"github.com/LeeSwallow/stickypane/internal/core/widget"
)

const (
	barWidth   = 20 // widest the progress bar gets
	formatHint = `no items yet: add "- [ ] task" lines`
)

// Checklist is the widget for a checklist note: the list and where the
// cursor is. The screen shows the open items first, in the order of the
// file, then the finished ones, the most recent first; the file's own order
// is never changed.
type Checklist struct {
	list
	cursor int           // index into order
	drawn  []widget.Span // the lines of each item in the last Draw, in order
	env    widget.Env
}

// order lists the items as the screen shows them, as indexes into items:
// the open ones, then the done ones from the most recent; those without a
// time keep the file's order after the dated ones.
func (c *Checklist) order() []int {
	var open, done []int
	for i, e := range c.items {
		if c.entries[e].checked {
			done = append(done, i)
		} else {
			open = append(open, i)
		}
	}
	sort.SliceStable(done, func(a, b int) bool {
		ta, tb := c.entries[c.items[done[a]]].done, c.entries[c.items[done[b]]].done
		if ta.IsZero() != tb.IsZero() {
			return !ta.IsZero()
		}
		return ta.After(tb)
	})
	return append(open, done...)
}

// selected is the entry under the cursor.
func (c *Checklist) selected() int { return c.items[c.order()[c.cursor]] }

// doneAt says when an item was done, as short as the day allows: the time
// for today, the day for this year, the date before that.
func (c *Checklist) doneAt(t time.Time) string { return c.env.Short(t) }

func (c *Checklist) bar(done, total, width int) string {
	label := fmt.Sprintf(" %d/%d", done, total)
	w := min(barWidth, width-widget.Width(label))
	if w < 1 {
		return widget.Truncate(strings.TrimSpace(label), width)
	}
	fill := done * w / total
	return c.env.Styles.Good.Render(strings.Repeat("▓", fill)) + c.env.Styles.Faint.Render(strings.Repeat("░", w-fill)) + label
}

// nth is how many items above the cursor share the selected item's text. It
// tells an Op which of several identical items is meant.
func (c *Checklist) nth() int {
	sel := c.selected()
	text := c.entries[sel].text
	n := 0
	for _, i := range c.items {
		if i == sel {
			break
		}
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
		lines := []string{c.env.Styles.Faint.Render(widget.Truncate(c.env.T(formatHint), width))}
		for _, e := range c.entries {
			if e.text != "" {
				lines = append(lines, widget.Wrap(widget.Clean(e.text), max(width, 1))...)
			}
		}
		return widget.Fit(strings.Join(lines, "\n"), width), widget.NoSpan
	}
	c.clamp()
	lines := []string{c.bar(done, total, width), ""}
	at := widget.NoSpan
	order := c.order()
	pos := make(map[int]int, len(order)) // entry -> place on the screen
	for p, i := range order {
		pos[c.items[i]] = p
	}
	c.drawn = make([]widget.Span, len(order))
	item := func(e entry, p int) {
		mark, style := "☐ ", func(s string) string { return s }
		if e.checked {
			// A finished item steps back so the open ones stand out.
			mark, style = "☑ ", func(s string) string { return c.env.Styles.Struck.Render(s) }
		}
		stamp := c.doneAt(e.done)
		textWidth := max(width-4, 1)
		if stamp != "" {
			textWidth = max(width-4-widget.Width(stamp)-1, 1)
		}
		selected := active && p == c.cursor
		from := len(lines)
		for j, l := range widget.Wrap(widget.Clean(widget.Unlink(e.text)), textWidth) {
			switch {
			case j == 0 && selected:
				at.Start = len(lines)
				l = c.env.Styles.Selected.Render("› " + mark + l)
			case j == 0:
				l = "  " + mark + style(l)
			case selected:
				l = c.env.Styles.Selected.Render("    " + l)
			default:
				l = "    " + style(l)
			}
			if j == 0 && stamp != "" {
				l += strings.Repeat(" ", max(width-widget.Width(l)-widget.Width(stamp), 1)) + c.env.Styles.Faint.Render(stamp)
			}
			lines = append(lines, l)
		}
		if selected {
			at.End = len(lines) // every wrapped line of the item
		}
		c.drawn[p] = widget.Span{Start: from, End: len(lines)}
	}
	// What is left, with the file's other lines where they are.
	for i, e := range c.entries {
		switch {
		case !e.item:
			for _, l := range widget.Wrap(widget.Clean(e.text), max(width, 1)) {
				if strings.HasPrefix(e.text, "#") {
					l = c.env.Styles.Bold.Render(l)
				}
				lines = append(lines, l)
			}
		case !e.checked:
			item(e, pos[i])
		}
	}
	// What is done, the most recent first.
	if done > 0 {
		if len(lines) > 2 {
			lines = append(lines, "")
		}
		lines = append(lines, c.env.Styles.Faint.Render(fmt.Sprintf(c.env.T("Done (%d)"), done)))
		for p := len(order) - done; p < len(order); p++ {
			item(c.entries[c.items[order[p]]], p)
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
			e := &c.entries[c.selected()]
			op := Toggle{Text: e.text, Checked: !e.checked, Nth: c.nth()}
			if op.Checked {
				op.At = Stamp(c.env.Now())
				e.done, _ = time.ParseInLocation(StampLayout, op.At, time.Local)
			} else {
				e.done = time.Time{}
			}
			res.Op = op
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
	nc := parse(d, c.env)

	nc.cursor = c.cursor
	nc.clamp()
	return nc
}
