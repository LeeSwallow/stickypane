// Package kinds bundles the built-in note shapes. To add a shape, create its
// package under internal/widget and add one line to Default.
package kinds

import (
	"charm.land/glamour/v2"

	"github.com/LeeSwallow/stickypane/internal/theme"
	"github.com/LeeSwallow/stickypane/internal/widget"
	"github.com/LeeSwallow/stickypane/internal/widget/board"
	"github.com/LeeSwallow/stickypane/internal/widget/chart"
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
		chart.Kind,
		form.NewKind(render),
		script.Kind,
	}
}

// Markdown returns the renderer for plain notes: Markdown styled with the
// theme in use at each call, with Mermaid blocks drawn as diagrams.
func Markdown(th *theme.Holder) note.Renderer { return WithMermaid(styled(th)) }

// styled renders Markdown with Glamour in the colors of the theme, and
// falls back to plain wrapping if Glamour fails.
func styled(th *theme.Holder) note.Renderer {
	type key struct {
		theme string
		width int
	}
	renderers := map[key]*glamour.TermRenderer{}
	return func(markdown string, width int) string {
		if width <= 0 {
			return markdown
		}
		t := th.Get()
		k := key{t.Name, width}
		r, ok := renderers[k]
		if !ok {
			if len(renderers) > 64 {
				clear(renderers)
			}
			var err error
			r, err = glamour.NewTermRenderer(glamour.WithStyles(t.Markdown()), glamour.WithWordWrap(width))
			if err != nil {
				return note.Plain(markdown, width)
			}
			renderers[k] = r
		}
		out, err := r.Render(joinListLines(markdown))
		if err != nil {
			return note.Plain(markdown, width)
		}
		return out
	}
}
