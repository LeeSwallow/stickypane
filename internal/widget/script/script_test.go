package script

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"

	"github.com/LeeSwallow/stickypane/internal/doc"
	"github.com/LeeSwallow/stickypane/internal/widget"
)

const deploy = "#!/bin/sh\n# ship it\ngo test ./... && ./deploy.sh\n"

func parseSrc(s string) widget.Widget { return Kind.Parse(doc.Document{Body: s}) }

func TestKind(t *testing.T) {
	if Kind.Name != "script" || Kind.New != ".sh" || strings.Join(Kind.Exts, " ") != ".sh .ps1" {
		t.Errorf("Kind = %+v", Kind)
	}
	if widget.Width(Kind.Icon) != 1 || Kind.Blurb == "" || Kind.Example == "" || !Kind.Handles("enter") || !Kind.Handles("space") {
		t.Errorf("Kind needs an icon, a blurb, an example and the keys that run it: %+v", Kind)
	}
	if got := string(Kind.Template("Deploy")); !strings.HasPrefix(got, "#!/bin/sh\n") || strings.Contains(got, "---") {
		t.Errorf("a new script is a shell script, without front matter: %q", got)
	}
}

func TestDrawShowsTheButtonAndTheScript(t *testing.T) {
	w := parseSrc(deploy)
	for _, width := range []int{1, 6, 20, 60} {
		out, at := w.Draw(width, true)
		for _, l := range strings.Split(out, "\n") {
			if got := widget.Width(l); got > width {
				t.Errorf("width %d: line %q is %d cells wide", width, ansi.Strip(l), got)
			}
		}
		if width >= 20 && (!at.Ok() || at.Start != 0) {
			t.Errorf("the button is what is selected: %+v", at)
		}
	}
	out, _ := w.Draw(60, true)
	text := ansi.Strip(out)
	for _, want := range []string{"Run", "1 #!/bin/sh", "2 # ship it", "3 go test ./... && ./deploy.sh"} {
		if !strings.Contains(text, want) {
			t.Errorf("Draw should contain %q:\n%s", want, text)
		}
	}
	if got := w.Summary(); got != "3 lines" {
		t.Errorf("Summary = %q", got)
	}
}

func TestEnterAsksToRun(t *testing.T) {
	w := parseSrc(deploy)
	for _, key := range []string{"enter", "space"} {
		if _, res := w.Update(key); !res.Run || res.Op != nil || res.Prompt != nil {
			t.Errorf("%s should ask the app to run the script: %+v", key, res)
		}
	}
	if _, res := w.Update("j"); res.Run {
		t.Error("other keys do not run anything")
	}
	c, ok := w.(widget.Clicker)
	if !ok {
		t.Fatal("the button can be clicked")
	}
	w.Draw(60, true)
	if _, res, hit := c.Click(1, 3); !hit || !res.Run {
		t.Errorf("a click on the button runs the script: %+v, %v", res, hit)
	}
	if _, res, hit := c.Click(5, 3); hit || res.Run {
		t.Errorf("a click on the script text does nothing: %+v, %v", res, hit)
	}
}

func TestAnEmptyScriptSaysSo(t *testing.T) {
	out, _ := parseSrc("").Draw(40, true)
	if !strings.Contains(ansi.Strip(out), "empty") {
		t.Errorf("Draw = %q", ansi.Strip(out))
	}
	if _, res := parseSrc("  \n").Update("enter"); res.Run {
		t.Error("there is nothing to run in an empty script")
	}
}
