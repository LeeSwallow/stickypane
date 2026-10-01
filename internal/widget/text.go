package widget

import (
	"strings"
	"unicode"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

// Shared text styles.
var (
	Bold     = lipgloss.NewStyle().Bold(true)
	Faint    = lipgloss.NewStyle().Faint(true)
	Selected = lipgloss.NewStyle().Reverse(true)
	Good     = lipgloss.NewStyle().Foreground(lipgloss.Color("#8FD694"))
	Warn     = lipgloss.NewStyle().Foreground(lipgloss.Color("#F5A962"))
	Struck   = lipgloss.NewStyle().Faint(true).Strikethrough(true)
)

// Clean prepares file text for the screen: escape sequences and control
// characters are dropped and tabs become four spaces. Newlines are kept.
func Clean(s string) string {
	s = strings.ReplaceAll(ansi.Strip(s), "\t", "    ")
	return strings.Map(func(r rune) rune {
		if r != '\n' && unicode.IsControl(r) {
			return -1
		}
		return r
	}, s)
}

// Width returns how many terminal cells s occupies.
func Width(s string) int { return ansi.StringWidth(s) }

// Truncate shortens s to at most w cells, ending with "…" when it was cut.
func Truncate(s string, w int) string {
	if w <= 0 {
		return ""
	}
	return ansi.Truncate(s, w, "…")
}

// Pad truncates or right-pads s to exactly w cells.
func Pad(s string, w int) string {
	if w <= 0 {
		return ""
	}
	s = Truncate(s, w)
	if gap := w - Width(s); gap > 0 {
		s += strings.Repeat(" ", gap)
	}
	return s
}

// Wrap breaks s into lines of at most w cells. Words longer than a line are
// split, so no line is ever wider than w.
func Wrap(s string, w int) []string {
	if w <= 0 {
		return []string{s}
	}
	return strings.Split(ansi.Wrap(s, w, ""), "\n")
}

// Fit cuts every line of s to at most w cells, without an ellipsis. It is the
// last safeguard for widths too small for the content, such as a pane
// narrower than one wide character.
func Fit(s string, w int) string {
	lines := strings.Split(s, "\n")
	for i, l := range lines {
		if Width(l) > w {
			lines[i] = ansi.Truncate(l, max(w, 0), "")
		}
	}
	return strings.Join(lines, "\n")
}

// ClampOffset keeps a scroll offset inside [0, total-height].
func ClampOffset(offset, total, height int) int {
	if m := total - height; offset > m {
		offset = m
	}
	if offset < 0 {
		offset = 0
	}
	return offset
}

// Window returns the lines visible at offset in an area of the given height.
func Window(lines []string, offset, height int) []string {
	if height <= 0 {
		return nil
	}
	offset = ClampOffset(offset, len(lines), height)
	return lines[offset:min(offset+height, len(lines))]
}

// ScrollKey returns the offset after a scrolling key and whether key was
// one. page is how many lines are in view: the paging keys move by that
// much, less one line of overlap. The result may exceed the content;
// ClampOffset fixes that when drawing.
func ScrollKey(offset int, key string, page int) (int, bool) {
	page = max(page-1, 1)
	switch key {
	case "j", "down":
		offset++
	case "k", "up":
		offset--
	case "space", "pgdown", "ctrl+f":
		offset += page
	case "b", "pgup", "ctrl+b":
		offset -= page
	case "ctrl+d":
		offset += max(page/2, 1)
	case "ctrl+u":
		offset -= max(page/2, 1)
	case "g":
		offset = 0
	case "G":
		offset = 1 << 30
	default:
		return offset, false
	}
	return max(offset, 0), true
}
