package app

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"

	"github.com/LeeSwallow/stickypane/internal/widget"
)

func TestFrameTitle(t *testing.T) {
	c := palette[0].color
	cases := []struct {
		title   string
		focused bool
		top     string
	}{
		{"Plan", false, "╭ Plan " + strings.Repeat("─", 12) + "╮"},
		{"", false, "╭" + strings.Repeat("─", 18) + "╮"},
		{"Plan", true, "╔ Plan " + strings.Repeat("═", 12) + "╗"},
	}
	for _, tc := range cases {
		lines := strings.Split(ansi.Strip(frame(tc.title, "x", 20, c, tc.focused)), "\n")
		if lines[0] != tc.top {
			t.Errorf("top = %q, want %q", lines[0], tc.top)
		}
		if len(lines) != 3 {
			t.Errorf("a one-line body should give 3 lines, got %d", len(lines))
		}
	}
}

func TestFrameKeepsWidthWithWideText(t *testing.T) {
	body := "토큰은 세션 쿠키로\nshort\n" + strings.Repeat("긴", 40)
	for _, focused := range []bool{false, true} {
		for _, title := range []string{"", "인증 설계 보고서", "📌 ● 아주 긴 제목입니다 정말로 길어서 잘려야 하는 제목"} {
			for _, line := range strings.Split(frame(title, body, 36, palette[1].color, focused), "\n") {
				if w := widget.Width(line); w != 36 {
					t.Errorf("title %q: line %q is %d cells wide, want 36", title, ansi.Strip(line), w)
				}
			}
		}
	}
}

func TestColorIndex(t *testing.T) {
	if got := colorIndex("a.md", "blue"); palette[got].name != "blue" {
		t.Errorf("a named color should win, got %q", palette[got].name)
	}
	if colorIndex("a.md", "BLUE") != colorIndex("a.md", "blue") {
		t.Error("color names are case-insensitive")
	}
	if colorIndex("a.md", "") != colorIndex("a.md", "no-such-color") {
		t.Error("an unknown color should fall back to the file name's color")
	}
	if colorIndex("a.md", "") != colorIndex("a.md", "") {
		t.Error("the fallback color must be stable")
	}
	if len(palette) != 6 {
		t.Errorf("palette has %d colors, want 6", len(palette))
	}
}
