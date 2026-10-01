package kinds

import (
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/AlexanderGrooff/mermaid-ascii/pkg/diagram"
	"github.com/AlexanderGrooff/mermaid-ascii/pkg/render"

	"github.com/LeeSwallow/stickypane/internal/widget"
	"github.com/LeeSwallow/stickypane/internal/widget/note"
)

const (
	// maxDiagramLines is the largest Mermaid block that is drawn. The
	// renderer's time and memory grow steeply with the number of edges, and
	// it runs while the screen is being drawn.
	maxDiagramLines = 80
	// maxDrawn is how many drawings are remembered.
	maxDrawn = 64
)

// errTooWide reports a diagram that needs more columns than the note has.
type errTooWide struct{ cells int }

func (e errTooWide) Error() string {
	return fmt.Sprintf("the diagram needs %d columns", e.cells)
}

// WithMermaid returns a renderer that draws Mermaid code blocks as diagrams
// and hands everything else to next. Agents often explain structure with a
// Mermaid block; in a terminal the source is noise and the picture is the
// point. A block that cannot be drawn (a diagram type the renderer does not
// know, a syntax error, a block that is never closed, one too large to draw
// quickly) is passed to next unchanged, so its source stays readable. A
// diagram wider than the note is not cut: its source is shown with a line
// saying how to see the drawing.
//
// Drawings are remembered by source and width, because a note is redrawn far
// more often than its diagrams change.
func WithMermaid(next note.Renderer) note.Renderer {
	type key struct {
		src   string
		width int
	}
	type drawn struct {
		out string
		err error
	}
	cache := map[key]drawn{}
	draw := func(src string, width int) (string, error) {
		k := key{src, width}
		if d, ok := cache[k]; ok {
			return d.out, d.err
		}
		if len(cache) >= maxDrawn {
			clear(cache)
		}
		out, err := drawMermaid(src, width)
		cache[k] = drawn{out, err}
		return out, err
	}
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
				drawing, err := draw(seg.text, width)
				if err == nil {
					flush()
					out = append(out, drawing)
					continue
				}
				prose.WriteString(seg.raw)
				var wide errTooWide
				if errors.As(err, &wide) {
					fmt.Fprintf(&prose, "\n_This diagram needs %d columns: zoom in or widen the pane to see it drawn._\n\n", wide.cells)
				}
				continue
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

// fence describes the line that opens a code block: the fence character,
// how many of them, and the word after them.
type fence struct {
	char byte
	n    int
	info string
}

// openFence reads a line that opens a code block.
func openFence(line string) (fence, bool) {
	line = strings.TrimSpace(line)
	if len(line) < 3 || (line[0] != '`' && line[0] != '~') {
		return fence{}, false
	}
	n := 0
	for n < len(line) && line[n] == line[0] {
		n++
	}
	if n < 3 {
		return fence{}, false
	}
	return fence{char: line[0], n: n, info: strings.TrimSpace(line[n:])}, true
}

// closes reports whether line closes the block f opened: the same character,
// at least as many of them, and nothing else.
func (f fence) closes(line string) bool {
	line = strings.TrimSpace(line)
	return len(line) >= f.n && strings.Trim(line, string(f.char)) == ""
}

// splitMermaid cuts markdown at Mermaid code blocks. Other code blocks are
// stepped over whole, so a Mermaid block quoted inside a longer fence stays
// the example it is. A block without a closing fence is left as prose.
func splitMermaid(markdown string) []segment {
	var segs []segment
	var prose strings.Builder
	lines := strings.SplitAfter(markdown, "\n")
	for i := 0; i < len(lines); i++ {
		f, ok := openFence(lines[i])
		end := -1
		if ok {
			for j := i + 1; j < len(lines); j++ {
				if f.closes(lines[j]) {
					end = j
					break
				}
			}
		}
		if end < 0 {
			prose.WriteString(lines[i])
			continue
		}
		if !strings.EqualFold(f.info, "mermaid") {
			prose.WriteString(strings.Join(lines[i:end+1], ""))
			i = end
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

var (
	// flowRe matches the first line of a flowchart and captures its direction.
	flowRe = regexp.MustCompile(`^(\s*(?:graph|flowchart)\s+)(LR|RL|TD|TB|BT)\b`)
	// labelledRe matches an arrow written "A -- text --> B".
	labelledRe = regexp.MustCompile(`--\s+([^-|>\n][^|>\n]*?)\s+-->`)
	// stylingRe matches statements that only change how a flowchart looks in
	// a browser. The text renderer would draw them as boxes.
	stylingRe = regexp.MustCompile(`^\s*(style|click|linkStyle)\s`)
)

// tidy prepares flowchart source for the text renderer: styling statements
// are dropped and "A -- text --> B" becomes "A -->|text| B". Other diagram
// types are returned as they are.
func tidy(src string) string {
	if !flowRe.MatchString(src) {
		return src
	}
	var out []string
	for _, line := range strings.Split(src, "\n") {
		if stylingRe.MatchString(line) {
			continue
		}
		out = append(out, labelledRe.ReplaceAllString(line, "-->|$1|"))
	}
	return strings.Join(out, "\n")
}

// drawMermaid renders Mermaid source as text no wider than width. A
// left-to-right flowchart that does not fit is drawn top-down instead.
func drawMermaid(src string, width int) (string, error) {
	lines := 0
	for _, l := range strings.Split(src, "\n") {
		if strings.TrimSpace(l) != "" {
			lines++
		}
	}
	switch {
	case lines == 0:
		return "", errors.New("empty diagram")
	case lines > maxDiagramLines:
		return "", fmt.Errorf("the diagram has %d lines, more than %d", lines, maxDiagramLines)
	}
	src = tidy(src)
	out, err := renderDiagram(src, width)
	if err != nil {
		return "", err
	}
	cells := widest(out)
	if cells <= width {
		return out, nil
	}
	if m := flowRe.FindStringSubmatch(src); m != nil && (m[2] == "LR" || m[2] == "RL") {
		down := flowRe.ReplaceAllString(src, "${1}TD")
		if out, err := renderDiagram(down, width); err == nil && widest(out) <= width {
			return out, nil
		}
	}
	return "", errTooWide{cells}
}

func widest(s string) int {
	w := 0
	for _, l := range strings.Split(s, "\n") {
		w = max(w, widget.Width(l))
	}
	return w
}

// renderDiagram runs the Mermaid renderer. It is a variable so tests can
// count its calls. The renderer is a parser for text an agent wrote, so a
// panic inside it is treated like any other failure to draw.
var renderDiagram = func(src string, width int) (out string, err error) {
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
