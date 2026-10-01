package kinds

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"

	"github.com/LeeSwallow/stickypane/internal/widget"
	"github.com/LeeSwallow/stickypane/internal/widget/note"
)

func TestDefaultRegistry(t *testing.T) {
	reg := Default(note.Plain)
	var names []string
	for _, k := range reg {
		names = append(names, k.Name)
		if k.Label == "" || k.Template == nil || k.Parse == nil {
			t.Errorf("kind %q is incomplete", k.Name)
		}
	}
	if got := strings.Join(names, ","); got != "note,board,checklist,log" {
		t.Errorf("kinds = %s", got)
	}
	if reg.Lookup("mystery").Name != "note" {
		t.Error("unknown types should fall back to the plain note")
	}
}

func TestMarkdownFollowsTheTheme(t *testing.T) {
	theme := &Theme{Dark: true}
	render := Markdown(theme)
	dark := render("## Heading\n\ntext\n", 30)
	theme.Dark = false
	light := render("## Heading\n\ntext\n", 30)
	if dark == light {
		t.Error("flipping the theme should change the colors of later renders")
	}
	if ansi.Strip(dark) != ansi.Strip(light) {
		t.Error("the theme must only change colors, not the text")
	}
}

func TestMarkdownRendersWithinWidth(t *testing.T) {
	for _, dark := range []bool{true, false} {
		out := Markdown(&Theme{Dark: dark})("## 결정 사항\n\n토큰은 세션 쿠키로 보관한다. **Important** item.\n\n- a\n- b\n", 30)
		plain := ansi.Strip(out)
		if !strings.Contains(plain, "결정 사항") || !strings.Contains(plain, "Important") {
			t.Errorf("rendered text lost content: %q", plain)
		}
		for _, line := range strings.Split(out, "\n") {
			if w := widget.Width(line); w > 30 {
				t.Errorf("line %q is %d cells wide, want at most 30", ansi.Strip(line), w)
			}
		}
	}
}
