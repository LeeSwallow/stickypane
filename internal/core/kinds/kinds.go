// Package kinds bundles the built-in note shapes. To add a shape, create its
// package under internal/core/widget and add one line to Default.
package kinds

import (
	"charm.land/lipgloss/v2"

	"github.com/LeeSwallow/stickypane/internal/core/markdown"
	"github.com/LeeSwallow/stickypane/internal/core/theme"
	"github.com/LeeSwallow/stickypane/internal/core/widget"
	"github.com/LeeSwallow/stickypane/internal/core/widget/api"
	"github.com/LeeSwallow/stickypane/internal/core/widget/board"
	"github.com/LeeSwallow/stickypane/internal/core/widget/chart"
	"github.com/LeeSwallow/stickypane/internal/core/widget/chat"
	"github.com/LeeSwallow/stickypane/internal/core/widget/checklist"
	"github.com/LeeSwallow/stickypane/internal/core/widget/form"
	"github.com/LeeSwallow/stickypane/internal/core/widget/html"
	"github.com/LeeSwallow/stickypane/internal/core/widget/logview"
	"github.com/LeeSwallow/stickypane/internal/core/widget/note"
	"github.com/LeeSwallow/stickypane/internal/core/widget/script"
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
		html.Kind,
	}
}

// Markdown returns the renderer for plain notes: Markdown styled with the
// theme in use at each call, with Mermaid blocks drawn as diagrams.
func Markdown(th *theme.Holder) note.Renderer { return WithEmbeds(WithMermaid(styled(th))) }

// styled renders Markdown in the colors of the theme in use at each call.
func styled(th *theme.Holder) note.Renderer {
	return func(src string, width int, _ widget.Env) string {
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
