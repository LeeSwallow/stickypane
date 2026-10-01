package app

import (
	"hash/fnv"
	"image/color"
	"strings"

	"charm.land/lipgloss/v2"

	"github.com/LeeSwallow/stickypane/internal/widget"
)

// palette is the set of note colors, in the order "c" cycles through them.
var palette = []struct {
	name  string
	color color.Color
}{
	{"yellow", lipgloss.Color("#F2D45C")},
	{"pink", lipgloss.Color("#F28FB1")},
	{"blue", lipgloss.Color("#7FB3F5")},
	{"green", lipgloss.Color("#8FD694")},
	{"purple", lipgloss.Color("#B79CF2")},
	{"orange", lipgloss.Color("#F5A962")},
}

// neutral is the color of frames that belong to the screen, not to a note.
var neutral = lipgloss.Color("#9AA5B1")

// colorIndex picks a note's color: the named one, or one derived from the
// file name so that a note keeps its color between runs.
func colorIndex(name, key string) int {
	for i, p := range palette {
		if strings.EqualFold(p.name, key) {
			return i
		}
	}
	h := fnv.New32a()
	h.Write([]byte(name))
	return int(h.Sum32() % uint32(len(palette)))
}

func noteColor(it item) color.Color {
	key, _ := it.note.Doc.Get("color")
	return palette[colorIndex(it.note.Name, key)].color
}

// box is everything a frame shows.
type box struct {
	title   string // in the top border, after the icon
	icon    string // one cell, before the title
	summary string // at the right end of the top border, such as "2/3"
	body    string
	width   int
	color   color.Color
	focused bool
}

// frame draws a box: a rounded border in its color, or a double border when
// focused, so focus is visible without color too. The top border carries the
// icon, the title and, when there is room, the summary. Every line is exactly
// width cells wide.
func frame(b box) string {
	width := max(b.width, 8)
	tl, tr, bl, br, hz, vt := "╭", "╮", "╰", "╯", "─", "│"
	if b.focused {
		tl, tr, bl, br, hz, vt = "╔", "╗", "╚", "╝", "═", "║"
	}
	st := lipgloss.NewStyle().Foreground(b.color)

	// The label is "icon title"; either part may be missing.
	label := strings.TrimSpace(b.icon + " " + b.title)
	summary := b.summary
	// Corners, a space on each side of the label, and at least one border
	// cell after it; the summary needs a space on each side as well.
	room := width - 5
	if summary != "" && widget.Width(label)+widget.Width(summary)+2 > room {
		summary = ""
	}
	if summary != "" {
		room -= widget.Width(summary) + 2
	}
	label = widget.Truncate(label, room)

	var top strings.Builder
	used := 1
	top.WriteString(st.Render(tl))
	if label != "" {
		top.WriteString(" " + st.Bold(true).Render(label) + " ")
		used += widget.Width(label) + 2
	}
	tail := tr
	if summary != "" {
		tail = " " + summary + " " + tr
	}
	fill := max(width-used-widget.Width(tail), 0)
	top.WriteString(st.Render(strings.Repeat(hz, fill)))
	if summary != "" {
		top.WriteString(" " + st.Faint(true).Render(summary) + " ")
	}
	top.WriteString(st.Render(tr))

	lines := []string{top.String()}
	for _, l := range strings.Split(b.body, "\n") {
		lines = append(lines, st.Render(vt)+" "+widget.Pad(l, width-4)+" "+st.Render(vt))
	}
	lines = append(lines, st.Render(bl+strings.Repeat(hz, width-2)+br))
	return strings.Join(lines, "\n")
}

// dialog frames body in a box boxW wide and centers it in an area w wide and
// h tall, a little above the middle. Without room for the frame it returns
// the body as it is, so small panes still show the content.
func dialog(title string, body []string, boxW, w, h int) []string {
	boxW = min(boxW, w)
	if boxW < 12 || h < len(body)+2 {
		return body
	}
	framed := strings.Split(frame(box{title: title, body: strings.Join(body, "\n"), width: boxW, color: neutral}), "\n")
	left := strings.Repeat(" ", (w-boxW)/2)
	out := make([]string, (h-len(framed))/3, h)
	for _, l := range framed {
		out = append(out, left+l)
	}
	return out
}

// hints draws the bottom line's key guide from alternating keys and labels:
// the key stands out and what it does steps back. The most important pairs
// come first; one that does not fit in width is left out whole, along with
// everything after it, so the line never ends in half a word.
func hints(width int, pairs ...string) string {
	var parts []string
	used := 0
	for i := 0; i+1 < len(pairs); i += 2 {
		w := widget.Width(pairs[i]) + 1 + widget.Width(pairs[i+1])
		if len(parts) > 0 {
			w += 2
		}
		if used+w > width {
			break
		}
		used += w
		parts = append(parts, widget.Bold.Render(pairs[i])+" "+widget.Faint.Render(pairs[i+1]))
	}
	return strings.Join(parts, "  ")
}
