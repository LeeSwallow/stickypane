// Package kinds bundles the built-in note shapes. To add a shape, create its
// package under internal/widget and add one line to Default.
package kinds

import (
	"charm.land/lipgloss/v2"

	"github.com/LeeSwallow/stickypane/internal/markdown"
	"github.com/LeeSwallow/stickypane/internal/theme"
	"github.com/LeeSwallow/stickypane/internal/widget"
	"github.com/LeeSwallow/stickypane/internal/widget/api"
	"github.com/LeeSwallow/stickypane/internal/widget/board"
	"github.com/LeeSwallow/stickypane/internal/widget/chart"
	"github.com/LeeSwallow/stickypane/internal/widget/chat"
	"github.com/LeeSwallow/stickypane/internal/widget/checklist"
	"github.com/LeeSwallow/stickypane/internal/widget/form"
	"github.com/LeeSwallow/stickypane/internal/widget/logview"
	"github.com/LeeSwallow/stickypane/internal/widget/note"
	"github.com/LeeSwallow/stickypane/internal/widget/script"
)

// Default returns the built-in kinds. The plain note comes first because it
// is the fallback for unknown types.
func Default(render note.Renderer) widget.Registry {
	return widget.Registry{
		note.NewKind(render),
		board.Kind,
		checklist.Kind,
		logview.Kind,
		chat.Kind,
		chart.Kind,
		form.NewKind(render),
		script.Kind,
		api.Kind,
	}
}

// Markdown returns the renderer for plain notes: Markdown styled with the
// theme in use at each call, with Mermaid blocks drawn as diagrams.
func Markdown(th *theme.Holder) note.Renderer { return WithEmbeds(WithMermaid(styled(th))) }

// styled renders Markdown in the colors of the theme in use at each call.
func styled(th *theme.Holder) note.Renderer {
	return func(src string, width int) string {
		if width <= 0 {
			return src
		}
		return markdown.Render(src, width, markdownStyles(th.Get()))
	}
}

// markdownStyles are a theme's colors for Markdown: headings in the info
// color, links and bullets in the accent, code in the warning color, and
// what steps back muted. The text keeps the terminal's own color.
func markdownStyles(t theme.Theme) markdown.Styles {
	fg := func(hex string) lipgloss.Style { return lipgloss.NewStyle().Foreground(lipgloss.Color(hex)) }
	return markdown.Styles{
		Heading: fg(t.Info).Bold(true),
		Minor:   fg(t.Muted),
		Muted:   fg(t.Muted),
		Link:    fg(t.Accent).Underline(true),
		Code:    fg(t.Warn),
		Mark:    fg(t.Accent),
		Strong:  lipgloss.NewStyle().Bold(true),
		Emph:    lipgloss.NewStyle().Italic(true),
	}
}
