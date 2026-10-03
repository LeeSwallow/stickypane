package markdown

import (
	"strings"
	"testing"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

var plain = Styles{}

func render(src string, width int) string { return ansi.Strip(Render(src, width, plain)) }

func lines(s string) []string { return strings.Split(s, "\n") }

func TestHeadingsAndParagraphs(t *testing.T) {
	got := render("# Title\n\nA paragraph that is written\nover two lines.\n\n###### small\n", 40)
	want := "Title\n\nA paragraph that is written over two\nlines.\n\nsmall"
	if got != want {
		t.Errorf("got\n%s\nwant\n%s", got, want)
	}
}

func TestListsNestAndContinue(t *testing.T) {
	src := "- one\n  continued\n- two\n  - inner\n- [ ] open\n- [x] done\n\n1. first\n2. second\n"
	got := render(src, 40)
	want := "• one continued\n• two\n  • inner\n☐ open\n☑ done\n\n1. first\n2. second"
	if got != want {
		t.Errorf("got\n%s\nwant\n%s", got, want)
	}
}

func TestALongItemWrapsUnderItsText(t *testing.T) {
	got := render("- a long item that wraps onto a second line\n", 20)
	for i, l := range lines(got)[1:] {
		if !strings.HasPrefix(l, "  ") {
			t.Errorf("line %d does not hang under the text: %q", i+1, l)
		}
	}
}

func TestCodeQuotesRulesAndTables(t *testing.T) {
	src := "```go\nfunc main() {\n  stays\n}\n```\n\n> quoted\n\n---\n\n| name | n |\n|---|---|\n| a | 1 |\n| long name | 22 |\n"
	got := render(src, 30)
	for _, want := range []string{"  func main() {", "    stays", "│ quoted", strings.Repeat("─", 30), "name       n", "long name  22"} {
		if !strings.Contains(got, want) {
			t.Errorf("want %q in\n%s", want, got)
		}
	}
}

func TestInlineMarksAreDrawnNotShown(t *testing.T) {
	got := render("**bold**, *it*, `code`, [a link](https://x.y), ![pic](p.png) and \\*stars\\*\n", 80)
	if got != "bold, it, code, a link, [pic] and *stars*" {
		t.Errorf("got %q", got)
	}
}

// Nothing is wider than the width, whatever it is.
func TestNothingIsWiderThanTheWidth(t *testing.T) {
	src := "## 결정 사항\n\n토큰은 세션 쿠키로 보관한다. **Important** item.\n\n- 아주 긴 한글 항목 이름입니다 정말로 길어서 여러 줄이 됩니다\n\n```\nverylonglinewithoutspacesthatgoesonandon\n```\n\n| a | b |\n|-|-|\n| 한글 칸 | 아주 긴 내용이 들어간 칸 |\n"
	for _, w := range []int{1, 8, 20, 30} {
		for _, l := range lines(Render(src, w, plain)) {
			if got := ansi.StringWidth(l); got > w {
				t.Errorf("width %d: %q is %d wide", w, ansi.Strip(l), got)
			}
		}
	}
}

// Styles only color; the text stays the same.
func TestStylesOnlyColor(t *testing.T) {
	colored := Styles{Heading: lipgloss.NewStyle().Bold(true), Code: lipgloss.NewStyle().Italic(true)}
	src := "# H\n\ntext `c`\n"
	if a, b := Render(src, 20, colored), Render(src, 20, plain); a == b || ansi.Strip(a) != ansi.Strip(b) {
		t.Errorf("styles should change only the colors:\n%q\n%q", a, b)
	}
}
