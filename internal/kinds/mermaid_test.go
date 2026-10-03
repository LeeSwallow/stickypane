package kinds

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"

	"github.com/LeeSwallow/stickypane/internal/theme"
	"github.com/LeeSwallow/stickypane/internal/widget"
	"github.com/LeeSwallow/stickypane/internal/widget/note"
)

const flow = "Here is the flow.\n\n```mermaid\ngraph LR\n  key[키 입력] --> app --> store\n```\n\nThat is all.\n"

func TestMermaidBlocksBecomeDiagrams(t *testing.T) {
	out := ansi.Strip(WithMermaid(note.Plain)(flow, 70))
	for _, want := range []string{"Here is the flow.", "That is all.", "키 입력", "app", "store", "┌", "►"} {
		if !strings.Contains(out, want) {
			t.Errorf("output should contain %q:\n%s", want, out)
		}
	}
	for _, gone := range []string{"```", "graph LR", "-->"} {
		if strings.Contains(out, gone) {
			t.Errorf("the Mermaid source %q should be replaced by the drawing:\n%s", gone, out)
		}
	}
	if strings.Index(out, "Here is the flow.") > strings.Index(out, "┌") || strings.Index(out, "┌") > strings.Index(out, "That is all.") {
		t.Errorf("the diagram should sit where its block was:\n%s", out)
	}
}

func TestSequenceDiagramsAreDrawn(t *testing.T) {
	out := ansi.Strip(WithMermaid(note.Plain)("```mermaid\nsequenceDiagram\n  Agent->>File: write note\n  File-->>Board: changed\n```\n", 70))
	for _, want := range []string{"Agent", "File", "Board", "write note", "changed", "┌"} {
		if !strings.Contains(out, want) {
			t.Errorf("output should contain %q:\n%s", want, out)
		}
	}
}

func TestEntityDiagramsAreDrawn(t *testing.T) {
	md := "```mermaid\nerDiagram\n  USER ||--o{ NOTE : writes\n  USER {\n    int id PK\n    string name\n  }\n```\n"
	out := ansi.Strip(WithMermaid(note.Plain)(md, 70))
	for _, want := range []string{"USER", "NOTE", "writes", "│ int", "│ PK │", "┌"} {
		if !strings.Contains(out, want) {
			t.Errorf("output should contain %q:\n%s", want, out)
		}
	}
	if strings.Contains(out, "erDiagram") {
		t.Errorf("the source should be replaced by the drawing:\n%s", out)
	}
}

func TestWhatCannotBeDrawnShowsItsSource(t *testing.T) {
	cases := []string{
		"```mermaid\npie title Pets\n  \"Dogs\" : 3\n```\n", // a diagram type the renderer does not know
		"```mermaid\ngraph LR\n  A -->\n",                   // a block that is never closed
		"```mermaid\n```\n",                                 // an empty block
	}
	for _, md := range cases {
		out := ansi.Strip(WithMermaid(note.Plain)(md, 60))
		first := strings.Fields(strings.Split(md, "\n")[1])
		if len(first) > 0 && !strings.Contains(out, first[0]) {
			t.Errorf("the source must stay visible when it cannot be drawn:\n in: %q\nout: %q", md, out)
		}
	}
}

func TestOtherCodeBlocksAreLeftAlone(t *testing.T) {
	md := "```go\nfunc main() {}\n```\n\n~~~mermaid\ngraph TD\n  A --> B\n~~~\n"
	out := ansi.Strip(WithMermaid(note.Plain)(md, 60))
	if !strings.Contains(out, "func main() {}") || !strings.Contains(out, "```go") {
		t.Errorf("a Go block is not Mermaid and must pass through:\n%s", out)
	}
	if strings.Contains(out, "graph TD") || !strings.Contains(out, "┌") {
		t.Errorf("a ~~~mermaid block should be drawn too:\n%s", out)
	}
}

func TestDiagramsFitTheWidthWhenTheyCan(t *testing.T) {
	out := WithMermaid(note.Plain)("```mermaid\ngraph LR\n  A --> B --> C --> D\n```\n", 44)
	for _, line := range strings.Split(out, "\n") {
		if w := widget.Width(line); w > 44 {
			t.Errorf("line %q is %d cells wide, want at most 44", ansi.Strip(line), w)
		}
	}
}

func TestMarkdownDrawsMermaid(t *testing.T) {
	out := ansi.Strip(Markdown(theme.NewHolder(theme.Pick("auto", true)))(flow, 70))
	if !strings.Contains(out, "┌") || strings.Contains(out, "graph LR") || !strings.Contains(out, "That is all.") {
		t.Errorf("the styled renderer should draw Mermaid blocks too:\n%s", out)
	}
}

func TestAHugeDiagramKeepsItsSource(t *testing.T) {
	var sb strings.Builder
	sb.WriteString("```mermaid\ngraph LR\n")
	for i := 0; i < 400; i++ {
		sb.WriteString("  A --> B\n")
	}
	sb.WriteString("```\n")
	out := ansi.Strip(WithMermaid(note.Plain)(sb.String(), 80))
	if !strings.Contains(out, "graph LR") || strings.Contains(out, "┌") {
		t.Errorf("a block too large to draw quickly should stay as source:\n%.200s", out)
	}
}

func TestADiagramTooWideForTheNoteIsNotCut(t *testing.T) {
	md := "```mermaid\nsequenceDiagram\n  Alice->>Bob: a rather long message for a narrow pane\n```\n"
	out := WithMermaid(note.Plain)(md, 30)
	for _, line := range strings.Split(out, "\n") {
		if w := widget.Width(line); w > 30 {
			t.Fatalf("line %q is %d cells wide, want at most 30", ansi.Strip(line), w)
		}
	}
	plain := ansi.Strip(out)
	if !strings.Contains(plain, "sequenceDiagram") || !strings.Contains(plain, "zoom") {
		t.Errorf("a diagram that does not fit should show its source and say how to see it:\n%s", plain)
	}
}

func TestAWideFlowchartTurnsDownwards(t *testing.T) {
	md := "```mermaid\ngraph LR\n  alpha --> beta --> gamma --> delta --> epsilon\n```\n"
	out := ansi.Strip(WithMermaid(note.Plain)(md, 32))
	if strings.Contains(out, "graph LR") || !strings.Contains(out, "▼") || !strings.Contains(out, "epsilon") {
		t.Errorf("a left-to-right chart that does not fit should be drawn top-down:\n%s", out)
	}
}

func TestLongerFencesAreRespected(t *testing.T) {
	md := "Example:\n\n````markdown\n```mermaid\ngraph LR\n  A --> B\n```\n````\n\nafter\n"
	out := ansi.Strip(WithMermaid(note.Plain)(md, 60))
	if strings.Contains(out, "┌") || !strings.Contains(out, "graph LR") || !strings.Contains(out, "after") {
		t.Errorf("a Mermaid block quoted inside a longer fence is an example, not a diagram:\n%s", out)
	}
}

func TestStylingLinesAndLabelledArrowsAreUnderstood(t *testing.T) {
	md := "```mermaid\ngraph LR\n  A -- yes --> B\n  style A fill:#f9f\n  click A href \"x\"\n```\n"
	out := ansi.Strip(WithMermaid(note.Plain)(md, 70))
	if strings.Contains(out, "style") || strings.Contains(out, "click") || strings.Contains(out, "A -- yes") || !strings.Contains(out, "yes") {
		t.Errorf("styling is dropped and the arrow keeps its label:\n%s", out)
	}
}

func TestTheSameDiagramIsDrawnOnce(t *testing.T) {
	calls := 0
	old := renderDiagram
	renderDiagram = func(src string, width int) (string, error) { calls++; return old(src, width) }
	defer func() { renderDiagram = old }()
	render := WithMermaid(note.Plain)
	for i := 0; i < 5; i++ {
		render(flow, 70)
	}
	if calls != 1 {
		t.Errorf("the diagram was drawn %d times for the same source and width", calls)
	}
}
