package checklist

import (
	"errors"
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"

	"github.com/LeeSwallow/stickypane/internal/doc"
	"github.com/LeeSwallow/stickypane/internal/widget"
)

const body = "## Login API\n- [x] Add endpoint\n- [x] Validate input\n- [ ] Write tests\n"

func parseBody(s string) widget.Widget { return Kind.Parse(doc.Document{Body: s}) }

func apply(t *testing.T, op doc.Op, body string) string {
	t.Helper()
	d, err := op.Apply(doc.Document{Body: body})
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}
	return d.Body
}

func TestPreviewShowsProgressAndOpenItems(t *testing.T) {
	got := ansi.Strip(parseBody(body).Preview(30))
	want := "▓▓▓▓▓▓▓▓▓▓▓▓▓░░░░░░░ 2/3\n☐ Write tests"
	if got != want {
		t.Errorf("Preview = %q, want %q", got, want)
	}
}

func TestPreviewWhenAllDone(t *testing.T) {
	got := ansi.Strip(parseBody("- [x] a\n- [X] b\n").Preview(30))
	if !strings.HasSuffix(got, " 2/2\n✓ all done") {
		t.Errorf("Preview = %q", got)
	}
}

func TestPreviewLimitsOpenItems(t *testing.T) {
	got := ansi.Strip(parseBody("- [ ] 1\n- [ ] 2\n- [ ] 3\n- [ ] 4\n- [ ] 5\n- [ ] 6\n").Preview(30))
	if !strings.HasSuffix(got, "☐ 4\n+2 more") {
		t.Errorf("Preview = %q", got)
	}
}

func TestPreviewWithoutItemsExplainsFormat(t *testing.T) {
	if got := ansi.Strip(parseBody("nothing here\n").Preview(40)); !strings.Contains(got, `"- [ ] task"`) {
		t.Errorf("Preview = %q", got)
	}
}

func TestPreviewFitsNarrowWidth(t *testing.T) {
	for _, line := range strings.Split(parseBody("- [ ] 아주 긴 한글 항목 이름입니다 정말로\n").Preview(12), "\n") {
		if w := widget.Width(line); w > 12 {
			t.Errorf("line %q is %d cells wide", ansi.Strip(line), w)
		}
	}
}

func TestToggle(t *testing.T) {
	got := apply(t, Toggle{Text: "Write tests", Checked: true}, body)
	if want := strings.Replace(body, "- [ ] Write tests", "- [x] Write tests", 1); got != want {
		t.Errorf("got %q", got)
	}
	got = apply(t, Toggle{Text: "Add endpoint", Checked: false}, body)
	if want := strings.Replace(body, "- [x] Add endpoint", "- [ ] Add endpoint", 1); got != want {
		t.Errorf("got %q", got)
	}
}

func TestToggleKeepsCRLF(t *testing.T) {
	got := apply(t, Toggle{Text: "a", Checked: true}, "  * [ ] a\r\n- [ ] b\r\n")
	if want := "  * [x] a\r\n- [ ] b\r\n"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestToggleConflicts(t *testing.T) {
	for _, op := range []doc.Op{
		Toggle{Text: "gone", Checked: true},
		Toggle{Text: "Add endpoint", Checked: true}, // already checked
	} {
		d, err := op.Apply(doc.Document{Body: body})
		if !errors.Is(err, doc.ErrConflict) || d.Body != body {
			t.Errorf("%+v: err = %v, body changed = %v", op, err, d.Body != body)
		}
	}
}

func TestAddItem(t *testing.T) {
	cases := []struct{ body, want string }{
		{"- [x] a\n- [ ] b\n\nnotes\n", "- [x] a\n- [ ] b\n- [ ] new\n\nnotes\n"},
		{"", "- [ ] new\n"},
		{"intro\n", "intro\n- [ ] new\n"},
		{"intro", "intro\n- [ ] new"},
		{"- [ ] a", "- [ ] a\n- [ ] new"},
		{"- [ ] a\r\n", "- [ ] a\r\n- [ ] new\r\n"},
	}
	for _, c := range cases {
		if got := apply(t, AddItem{Text: "new"}, c.body); got != c.want {
			t.Errorf("add to %q: got %q, want %q", c.body, got, c.want)
		}
	}
}

func TestSpaceTogglesItemUnderCursor(t *testing.T) {
	w := parseBody(body)
	w, _ = w.Update("j")
	w, _ = w.Update("j")
	w, res := w.Update("space")
	if want := (Toggle{Text: "Write tests", Checked: true}); res.Op != want {
		t.Fatalf("Op = %+v, want %+v", res.Op, want)
	}
	if view := ansi.Strip(w.View(40, 10)); !strings.Contains(view, "› ☑ Write tests") {
		t.Errorf("the item should look checked right away:\n%s", view)
	}
}

func TestNewItemPrompt(t *testing.T) {
	_, res := parseBody(body).Update("n")
	if res.Prompt == nil || res.Prompt.Label != "New item" {
		t.Fatalf("Prompt = %+v", res.Prompt)
	}
	if got, want := res.Prompt.Submit("deploy"), (AddItem{Text: "deploy"}); got != want {
		t.Errorf("Submit = %+v", got)
	}
}

func TestViewKeepsCursorVisible(t *testing.T) {
	var sb strings.Builder
	for i := 0; i < 20; i++ {
		sb.WriteString("- [ ] item ")
		sb.WriteString(strings.Repeat("x", i+1))
		sb.WriteString("\n")
	}
	w := parseBody(sb.String())
	for i := 0; i < 19; i++ {
		w, _ = w.Update("j")
	}
	view := ansi.Strip(w.View(40, 5))
	if !strings.Contains(view, "› ☐ item "+strings.Repeat("x", 20)) {
		t.Errorf("cursor line is not visible:\n%s", view)
	}
	if n := strings.Count(view, "\n") + 1; n > 5 {
		t.Errorf("View is %d lines tall, want at most 5", n)
	}
}

func TestSyncClampsCursor(t *testing.T) {
	w := parseBody(body)
	w, _ = w.Update("j")
	w, _ = w.Update("j")
	w = w.Sync(doc.Document{Body: "- [ ] only\n"})
	if view := ansi.Strip(w.View(40, 5)); !strings.Contains(view, "› ☐ only") {
		t.Errorf("View = %q", view)
	}
}

func TestKindTemplate(t *testing.T) {
	if got := string(Kind.Template("Release")); got != "---\ntype: checklist\ntitle: Release\n---\n" {
		t.Errorf("Template = %q", got)
	}
}
