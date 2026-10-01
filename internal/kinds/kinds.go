// Package kinds bundles the built-in note shapes. To add a shape, create its
// package under internal/widget and add one line to Default.
package kinds

import (
	"charm.land/glamour/v2"
	"charm.land/glamour/v2/styles"

	"github.com/LeeSwallow/stickypane/internal/widget"
	"github.com/LeeSwallow/stickypane/internal/widget/board"
	"github.com/LeeSwallow/stickypane/internal/widget/chart"
	"github.com/LeeSwallow/stickypane/internal/widget/checklist"
	"github.com/LeeSwallow/stickypane/internal/widget/logview"
	"github.com/LeeSwallow/stickypane/internal/widget/note"
)

// Default returns the built-in kinds. The plain note comes first because it
// is the fallback for unknown types.
func Default(render note.Renderer) widget.Registry {
	return widget.Registry{
		note.NewKind(render),
		board.Kind,
		checklist.Kind,
		logview.Kind,
		chart.Kind,
	}
}

// Theme tells the Markdown renderer which colors suit the terminal. The app
// flips Dark when the terminal reports its background color, so nothing has
// to block on a terminal query before the first draw.
type Theme struct{ Dark bool }

// Markdown returns the renderer for plain notes: Markdown styled for the
// theme as it is at each call, with Mermaid blocks drawn as diagrams.
func Markdown(theme *Theme) note.Renderer { return WithMermaid(styled(theme)) }

// styled renders Markdown with Glamour. It drops Glamour's page margins so
// short notes stay compact, and falls back to plain wrapping if Glamour fails.
func styled(theme *Theme) note.Renderer {
	type key struct {
		dark  bool
		width int
	}
	renderers := map[key]*glamour.TermRenderer{}
	return func(markdown string, width int) string {
		if width <= 0 {
			return markdown
		}
		k := key{theme.Dark, width}
		r, ok := renderers[k]
		if !ok {
			cfg := styles.LightStyleConfig
			if theme.Dark {
				cfg = styles.DarkStyleConfig
			}
			var zero uint
			cfg.Document.Margin = &zero
			cfg.Document.BlockPrefix = ""
			cfg.Document.BlockSuffix = ""
			var err error
			r, err = glamour.NewTermRenderer(glamour.WithStyles(cfg), glamour.WithWordWrap(width))
			if err != nil {
				return note.Plain(markdown, width)
			}
			renderers[k] = r
		}
		out, err := r.Render(markdown)
		if err != nil {
			return note.Plain(markdown, width)
		}
		return out
	}
}
