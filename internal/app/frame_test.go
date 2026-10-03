package app

import (
	"charm.land/lipgloss/v2"

	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"

	"github.com/LeeSwallow/stickypane/internal/widget"
)

func top(b box) string {
	return strings.Split(ansi.Strip(frame(b)), "\n")[0]
}

func TestFrameTopBorder(t *testing.T) {
	cases := []struct {
		name string
		b    box
		want string
	}{
		{"title only", box{title: "Plan", width: 20}, "╭ Plan " + strings.Repeat("─", 12) + "╮"},
		{"nothing", box{width: 20}, "╭" + strings.Repeat("─", 18) + "╮"},
		{"focused", box{title: "Plan", width: 20, focused: true}, "╔ Plan " + strings.Repeat("═", 12) + "╗"},
		{"icon and title", box{title: "Plan", icon: "▦", width: 20}, "╭ ▦ Plan " + strings.Repeat("─", 10) + "╮"},
		{"icon alone", box{icon: "✎", width: 20}, "╭ ✎ " + strings.Repeat("─", 15) + "╮"},
		{"summary on the right", box{title: "Plan", icon: "▦", summary: "3 cards", width: 30}, "╭ ▦ Plan " + strings.Repeat("─", 11) + " 3 cards ╮"},
		{"no room for the summary", box{title: "A long title here", summary: "12 cards", width: 24}, "╭ A long title here ───╮"},
	}
	for _, c := range cases {
		c.b.body = "x"
		c.b.color = lipgloss.Color("#F2D45C")
		if got := top(c.b); got != c.want {
			t.Errorf("%s:\n got %q\nwant %q", c.name, got, c.want)
		}
	}
}

func TestFrameKeepsWidthWithWideText(t *testing.T) {
	body := "토큰은 세션 쿠키로\nshort\n" + strings.Repeat("긴", 40)
	for _, focused := range []bool{false, true} {
		for _, title := range []string{"", "인증 설계 보고서", "아주 긴 제목입니다 정말로 길어서 잘려야 하는 제목"} {
			for _, summary := range []string{"", "23 cards", "요약도 아주 길어서 들어갈 자리가 없습니다"} {
				b := box{title: title, icon: "▦", summary: summary, body: body, width: 36, color: lipgloss.Color("#F28FB1"), focused: focused}
				for _, line := range strings.Split(frame(b), "\n") {
					if w := widget.Width(line); w != 36 {
						t.Errorf("title %q, summary %q: line %q is %d cells wide, want 36", title, summary, ansi.Strip(line), w)
					}
				}
			}
		}
	}
}

func TestFrameHeightFollowsTheBody(t *testing.T) {
	if n := strings.Count(frame(box{body: "a\nb\nc", width: 20, color: lipgloss.Color("#F2D45C")}), "\n") + 1; n != 5 {
		t.Errorf("three body lines should give 5 lines, got %d", n)
	}
}

func TestDialogIsCenteredAndFallsBackWhenThereIsNoRoom(t *testing.T) {
	lines := dialog("Keys", []string{"one", "two"}, 20, 60, 12, lipgloss.Color("#7FB3F5"))
	var first string
	for _, l := range lines {
		if strings.TrimSpace(ansi.Strip(l)) != "" {
			first = ansi.Strip(l)
			break
		}
	}
	if !strings.HasPrefix(first, strings.Repeat(" ", 20)+"╔ Keys ") {
		t.Errorf("the dialog should be centered in 60 cells: %q", first)
	}
	for _, l := range lines {
		if w := widget.Width(l); w > 60 {
			t.Errorf("line is %d cells wide", w)
		}
	}
	if got := dialog("Keys", []string{"one", "two"}, 20, 10, 12, lipgloss.Color("#7FB3F5")); len(got) != 2 || got[0] != "one" {
		t.Errorf("without room for a frame the body should come back as it is: %q", got)
	}
	if got := dialog("Keys", []string{"one", "two"}, 20, 60, 3, lipgloss.Color("#7FB3F5")); len(got) != 2 {
		t.Errorf("without height for a frame the body should come back as it is: %q", got)
	}
}

func TestHintsPairKeysWithLabels(t *testing.T) {
	if got := ansi.Strip(hints(80, "tab", "next", "o", "close")); got != "tab next  o close" {
		t.Errorf("hints = %q", got)
	}
	if got := hints(80); got != "" {
		t.Errorf("no pairs should give an empty line, got %q", got)
	}
	// A pair that does not fit is left out whole, never cut in the middle.
	if got := ansi.Strip(hints(16, "tab", "next", "o", "close", "?", "help")); got != "tab next" {
		t.Errorf("hints in 16 cells = %q, want %q", got, "tab next")
	}
	if got := ansi.Strip(hints(17, "tab", "next", "o", "close", "?", "help")); got != "tab next  o close" {
		t.Errorf("hints in 17 cells = %q", got)
	}
}

func TestColorIndex(t *testing.T) {
	if got := colorIndex("a.md", "blue"); palette[got] != "blue" {
		t.Errorf("a named color should win, got %q", palette[got])
	}
	if colorIndex("a.md", "BLUE") != colorIndex("a.md", "blue") {
		t.Error("color names are case-insensitive")
	}
	if colorIndex("a.md", "") != colorIndex("a.md", "no-such-color") {
		t.Error("an unknown color should fall back to the file name's color")
	}
	if len(palette) != 6 {
		t.Errorf("palette has %d colors, want 6", len(palette))
	}
}
