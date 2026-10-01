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

// frame draws a note: a rounded border in the note's color, or a double
// border when focused, so focus is visible without color too. The title sits
// in the top border. Every line is exactly width cells wide.
func frame(title, body string, width int, c color.Color, focused bool) string {
	width = max(width, 8)
	tl, tr, bl, br, hz, vt := "╭", "╮", "╰", "╯", "─", "│"
	if focused {
		tl, tr, bl, br, hz, vt = "╔", "╗", "╚", "╝", "═", "║"
	}
	st := lipgloss.NewStyle().Foreground(c)

	top := st.Render(tl + strings.Repeat(hz, width-2) + tr)
	if title != "" {
		t := widget.Truncate(title, width-5)
		top = st.Render(tl+" ") + st.Bold(true).Render(t) +
			st.Render(" "+strings.Repeat(hz, width-4-widget.Width(t))+tr)
	}
	lines := []string{top}
	for _, l := range strings.Split(body, "\n") {
		lines = append(lines, st.Render(vt)+" "+widget.Pad(l, width-4)+" "+st.Render(vt))
	}
	lines = append(lines, st.Render(bl+strings.Repeat(hz, width-2)+br))
	return strings.Join(lines, "\n")
}
