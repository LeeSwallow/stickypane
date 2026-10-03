package kinds

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"

	"github.com/LeeSwallow/stickypane/internal/doc"
	"github.com/LeeSwallow/stickypane/internal/store"
	"github.com/LeeSwallow/stickypane/internal/theme"
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
	if got := strings.Join(names, ","); got != "note,board,checklist,log,chart,form,script" {
		t.Errorf("kinds = %s", got)
	}
	if reg.Lookup("script").Name != "note" || reg.For("run.sh", doc.Document{}).Name != "script" || reg.For("build.log", doc.Document{}).Name != "log" {
		t.Error("a script or a log is told by its file name, not by front matter")
	}
	if reg.Lookup("mystery").Name != "note" {
		t.Error("unknown types should fall back to the plain note")
	}
}

func TestMarkdownFollowsTheTheme(t *testing.T) {
	th := theme.NewHolder(theme.Pick("auto", true))
	render := Markdown(th)
	dark := render("## Heading\n\ntext\n", 30)
	th.Set(theme.Pick("auto", false))
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
		out := Markdown(theme.NewHolder(theme.Pick("auto", dark)))("## 결정 사항\n\n토큰은 세션 쿠키로 보관한다. **Important** item.\n\n- a\n- b\n", 30)
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

// A list item written over two lines is one item: its second line is not
// drawn as a line of its own at the left edge.
func TestAListItemOverTwoLinesStaysOneItem(t *testing.T) {
	r := Markdown(theme.NewHolder(theme.Pick("stickypane-dark", true)))
	out := ansi.Strip(r("- **e** edits a note, **D** deletes it and\n  **u** takes that back\n- next\n\n```\n- code\n  stays\n```\n", 60))
	if !strings.Contains(out, "deletes it and u takes that back") {
		t.Errorf("the second line should continue the item:\n%s", out)
	}
	if !strings.Contains(out, "  stays") {
		t.Errorf("a code block keeps its lines:\n%s", out)
	}
}

// A PowerShell script is a script note like a shell script: shown with a
// Run button, run by the interpreter the system has.
func TestAPowerShellScriptIsAScript(t *testing.T) {
	reg := Default(note.Plain)
	for _, name := range []string{"deploy.sh", "deploy.ps1", "Deploy.PS1"} {
		if k := reg.For(name, doc.Document{}); k.Name != "script" {
			t.Errorf("%s is a %s", name, k.Name)
		}
	}
	if !store.IsNote("deploy.ps1") {
		t.Error("the board shows .ps1 files")
	}
}
