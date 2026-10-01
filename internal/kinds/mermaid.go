package kinds

import (
	"errors"
	"fmt"
	"strings"

	"github.com/AlexanderGrooff/mermaid-ascii/pkg/diagram"
	"github.com/AlexanderGrooff/mermaid-ascii/pkg/render"

	"github.com/LeeSwallow/stickypane/internal/widget/note"
)

// WithMermaid returns a renderer that draws Mermaid code blocks as diagrams
// and hands everything else to next. Agents often explain structure with a
// Mermaid block; in a terminal the source is noise and the picture is the
// point. A block that cannot be drawn (a diagram type the renderer does not
// know, a syntax error, a block that is never closed) is passed to next
// unchanged, so its source stays readable.
func WithMermaid(next note.Renderer) note.Renderer {
	return func(markdown string, width int) string {
		var out []string
		var prose strings.Builder
		flush := func() {
			if strings.TrimSpace(prose.String()) != "" {
				out = append(out, strings.Trim(next(prose.String(), width), "\n"))
			}
			prose.Reset()
		}
		for _, seg := range splitMermaid(markdown) {
			if seg.mermaid {
				if drawing, err := drawMermaid(seg.text, width); err == nil {
					flush()
					out = append(out, drawing)
					continue
				}
			}
			prose.WriteString(seg.raw)
		}
		flush()
		return strings.Join(out, "\n\n")
	}
}

// segment is a piece of a Markdown document: prose, or one Mermaid block.
type segment struct {
	raw     string // the piece as written, fences included
	text    string // for a Mermaid block, the source between the fences
	mermaid bool
}

// splitMermaid cuts markdown at Mermaid code blocks fenced with ``` or ~~~.
// A block without a closing fence is left as prose.
func splitMermaid(markdown string) []segment {
	var segs []segment
	var prose strings.Builder
	lines := strings.SplitAfter(markdown, "\n")
	for i := 0; i < len(lines); i++ {
		fence := openFence(lines[i])
		end := -1
		if fence != "" {
			for j := i + 1; j < len(lines); j++ {
				if strings.TrimSpace(lines[j]) == fence {
					end = j
					break
				}
			}
		}
		if end < 0 {
			prose.WriteString(lines[i])
			continue
		}
		if prose.Len() > 0 {
			segs = append(segs, segment{raw: prose.String()})
			prose.Reset()
		}
		segs = append(segs, segment{
			raw:     strings.Join(lines[i:end+1], ""),
			text:    strings.Join(lines[i+1:end], ""),
			mermaid: true,
		})
		i = end
	}
	if prose.Len() > 0 {
		segs = append(segs, segment{raw: prose.String()})
	}
	return segs
}

// openFence returns the fence ("```" or "~~~") when line opens a Mermaid
// block, and "" otherwise.
func openFence(line string) string {
	line = strings.TrimSpace(line)
	for _, fence := range []string{"```", "~~~"} {
		if rest, ok := strings.CutPrefix(line, fence); ok && strings.EqualFold(strings.TrimSpace(rest), "mermaid") {
			return fence
		}
	}
	return ""
}

// drawMermaid renders Mermaid source as text, fitted to width when the
// renderer can. The renderer is a parser for text an agent wrote, so a panic
// inside it is treated like any other failure to draw.
func drawMermaid(src string, width int) (out string, err error) {
	if strings.TrimSpace(src) == "" {
		return "", errors.New("empty diagram")
	}
	defer func() {
		if r := recover(); r != nil {
			out, err = "", fmt.Errorf("mermaid: %v", r)
		}
	}()
	cfg := diagram.DefaultConfig()
	cfg.MaxWidth = max(width, 0)
	drawing, err := render.RenderDiagram(src, cfg)
	if err != nil {
		return "", err
	}
	lines := strings.Split(drawing, "\n")
	for i, l := range lines {
		lines[i] = strings.TrimRight(l, " ")
	}
	for len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	if len(lines) == 0 {
		return "", errors.New("empty diagram")
	}
	return strings.Join(lines, "\n"), nil
}
