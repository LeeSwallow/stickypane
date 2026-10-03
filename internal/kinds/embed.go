package kinds

import (
	"regexp"
	"strings"

	"github.com/LeeSwallow/stickypane/internal/widget"
	"github.com/LeeSwallow/stickypane/internal/widget/note"
)

// Embeds finds the note a ![[name]] line shows: its title and its widget,
// drawn read only. The screen sets it on every reload; without it an embed
// is drawn as a link.
var Embeds func(name string) (title string, w widget.Widget, ok bool)

// embedRe is a line that is only an embed: ![[name]], ![[name#part]] or
// ![[name|alias]].
var embedRe = regexp.MustCompile(`^\s*!\[\[([^\]|#]+)(?:#[^\]|]*)?(?:\|([^\]]*))?\]\]\s*$`)

// WithEmbeds draws each line that is only ![[name]] as that note, in its own
// shape and read only, under a line with its title; the rest goes to next.
// Embeds go one level deep: a note drawn inside another shows its own
// embeds as links, so a note that embeds itself cannot loop.
func WithEmbeds(next note.Renderer) note.Renderer {
	depth := 0
	return func(src string, width int) string {
		if Embeds == nil || depth > 0 || !strings.Contains(src, "![[") {
			return next(src, width)
		}
		var out []string
		var prose []string
		fence := ""
		flush := func() {
			if text := strings.Join(prose, "\n"); strings.TrimSpace(text) != "" {
				out = append(out, strings.Trim(next(text+"\n", width), "\n"))
			}
			prose = prose[:0]
		}
		for _, line := range strings.Split(src, "\n") {
			trimmed := strings.TrimSpace(line)
			if fence != "" {
				if strings.HasPrefix(trimmed, fence) {
					fence = ""
				}
				prose = append(prose, line)
				continue
			}
			if strings.HasPrefix(trimmed, "```") || strings.HasPrefix(trimmed, "~~~") {
				fence = trimmed[:3]
				prose = append(prose, line)
				continue
			}
			m := embedRe.FindStringSubmatch(line)
			if m == nil {
				prose = append(prose, line)
				continue
			}
			depth++
			title, w, ok := Embeds(strings.TrimSpace(m[1]))
			if ok {
				if alias := strings.TrimSpace(m[2]); alias != "" {
					title = alias
				}
				flush()
				drawn, _ := w.Draw(max(width-2, 1), false)
				block := []string{widget.Faint.Render(widget.Truncate("↳ "+widget.Clean(title), width))}
				for _, l := range strings.Split(drawn, "\n") {
					block = append(block, widget.Faint.Render("│ ")+l)
				}
				out = append(out, strings.Join(block, "\n"))
			} else {
				prose = append(prose, line)
			}
			depth--
		}
		flush()
		return strings.Join(out, "\n\n")
	}
}
