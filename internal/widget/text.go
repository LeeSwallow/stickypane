package widget

import (
	"hash/fnv"
	"regexp"
	"strings"
	"unicode"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/LeeSwallow/stickypane/internal/theme"
)

// Shared text styles, by role. Apply fills them from a theme; until then
// they hold the default dark theme.
var (
	Bold     lipgloss.Style
	Faint    lipgloss.Style // hints, details, what steps back
	Selected lipgloss.Style // the selected line
	Accent   lipgloss.Style // focus and buttons
	Good     lipgloss.Style
	Warn     lipgloss.Style
	Bad      lipgloss.Style
	Info     lipgloss.Style
	Struck   lipgloss.Style // done items
)

func init() { Apply(theme.Pick("auto", true).Styles()) }

// Translate turns a phrase a widget shows into the user's language. The
// app sets it; until then phrases stay as written.
var Translate = func(phrase string) string { return phrase }

// T translates a phrase a widget shows.
func T(phrase string) string { return Translate(phrase) }

// Apply makes every widget draw with the styles of a theme from now on.
func Apply(s theme.Styles) {
	Bold, Faint, Selected, Accent = s.Bold, s.Faint, s.Selected, s.Accent
	Good, Warn, Bad, Info, Struck = s.Good, s.Warn, s.Bad, s.Info, s.Struck
}

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

// noteLink is a link to another note: [[name]], [[name#item]], [[name|alias]].
var noteLink = regexp.MustCompile(`!?\[\[([^\]|]+)(?:\|([^\]]*))?\]\]`)

// Unlink writes a line's links to other notes as their alias or name, for
// widgets that draw text without Markdown (checklist items, cards).
func Unlink(s string) string {
	return noteLink.ReplaceAllStringFunc(s, func(m string) string {
		p := noteLink.FindStringSubmatch(m)
		if strings.TrimSpace(p[2]) != "" {
			return strings.TrimSpace(p[2])
		}
		return strings.TrimSpace(p[1])
	})
}

// NameStyle gives a name (an @owner, a chat's author) its color, the same
// each time it appears and in every note.
func NameStyle(name string) lipgloss.Style {
	styles := []lipgloss.Style{Info, Good, Accent, Warn}
	h := fnv.New32a()
	h.Write([]byte(name))
	return styles[h.Sum32()%uint32(len(styles))]
}
