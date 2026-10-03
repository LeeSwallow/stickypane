package checklist

import (
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/x/ansi"

	"github.com/LeeSwallow/stickypane/internal/doc"
	"github.com/LeeSwallow/stickypane/internal/widget"
)

const body = "## Login API\n- [x] Add endpoint\n- [x] Validate input\n- [ ] Write tests\n"

const sameItems = "## Backend\n- [ ] tests\n## Frontend\n- [ ] tests\n"

func parseBody(s string) widget.Widget { return Kind.Parse(doc.Document{Body: s}) }

func draw(w widget.Widget, width int, active bool) ([]string, int) {
	out, at := w.Draw(width, active)
	return strings.Split(ansi.Strip(out), "\n"), at.Start
}

func TestSpanCoversEveryLineOfAWrappedItem(t *testing.T) {
	w := parseBody("- [ ] an item that is long enough to wrap onto more lines than one\n- [ ] next\n")
	out, at := w.Draw(20, true)
	lines := strings.Split(ansi.Strip(out), "\n")
	if !at.Ok() || at.End-at.Start < 2 {
		t.Fatalf("span %+v should cover the wrapped item: %q", at, lines)
	}
	if block := strings.Join(lines[at.Start:at.End], " "); !strings.Contains(block, "than one") || strings.Contains(block, "next") {
		t.Errorf("span covers %q", block)
	}
}

func apply(t *testing.T, op doc.Op, body string) string {
	t.Helper()
	d, err := op.Apply(doc.Document{Body: body})
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}
	return d.Body
}

func TestDrawShowsProgressAndEveryLine(t *testing.T) {
	lines, cursor := draw(parseBody(body), 30, false)
	want := []string{
		"▓▓▓▓▓▓▓▓▓▓▓▓▓░░░░░░░ 2/3",
		"",
		"## Login API",
		"  ☐ Write tests",
		"",
		"Done (2)",
		"  ☑ Add endpoint",
		"  ☑ Validate input",
	}
	if strings.Join(lines, "\n") != strings.Join(want, "\n") {
		t.Errorf("Draw:\n%s\nwant:\n%s", strings.Join(lines, "\n"), strings.Join(want, "\n"))
	}
	if cursor != -1 {
		t.Errorf("cursor = %d, want -1 when inactive", cursor)
	}
}

func TestDrawDoesNotFoldItems(t *testing.T) {
	var sb strings.Builder
	for i := 1; i <= 12; i++ {
		fmt.Fprintf(&sb, "- [ ] item %02d\n", i)
	}
	lines, _ := draw(parseBody(sb.String()), 40, false)
	text := strings.Join(lines, "\n")
	if strings.Contains(text, "more") || !strings.Contains(text, "item 01") || !strings.Contains(text, "item 12") {
		t.Errorf("every item should be visible:\n%s", text)
	}
}

func TestDrawWrapsLongItems(t *testing.T) {
	lines, _ := draw(parseBody("- [ ] 아주 긴 한글 항목 이름입니다 정말로 길어서 여러 줄이 됩니다\n- [ ] short\n"), 20, true)
	for _, line := range lines {
		if w := widget.Width(line); w > 20 {
			t.Errorf("line %q is %d cells wide", line, w)
		}
	}
	joined := strings.Join(strings.Fields(strings.Join(lines, " ")), " ")
	if !strings.Contains(joined, "여러 줄이 됩니다") && !strings.Contains(strings.ReplaceAll(joined, " ", ""), "여러줄이됩니다") {
		t.Errorf("wrapping lost the end of the item: %q", lines)
	}
}

func TestCursorShowsOnlyWhenActive(t *testing.T) {
	w := parseBody(body)
	w, _ = w.Update("j")
	lines, cursor := draw(w, 40, true)
	// The open item comes first on the screen, so j reaches the done ones.
	if cursor < 0 || lines[cursor] != "› ☑ Add endpoint" {
		t.Errorf("cursor = %d, lines = %q", cursor, lines)
	}
	if lines, cursor := draw(w, 40, false); cursor != -1 || strings.Contains(strings.Join(lines, "\n"), "›") {
		t.Errorf("an inactive checklist must not show a cursor: %q", lines)
	}
}

func TestChecklistWithoutItemsStillShowsItsText(t *testing.T) {
	lines, cursor := draw(parseBody("# Steps\n1. [ ] one\n[ ] another\n"), 50, true)
	text := strings.Join(lines, "\n")
	for _, want := range []string{`"- [ ] task"`, "# Steps", "1. [ ] one", "[ ] another"} {
		if !strings.Contains(text, want) {
			t.Errorf("Draw should contain %q:\n%s", want, text)
		}
	}
	if cursor != -1 {
		t.Errorf("cursor = %d, want -1 without items", cursor)
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
		Toggle{Text: "tests", Checked: true, Nth: 2},
	} {
		d, err := op.Apply(doc.Document{Body: body})
		if !errors.Is(err, doc.ErrConflict) || d.Body != body {
			t.Errorf("%+v: err = %v, body changed = %v", op, err, d.Body != body)
		}
	}
}

func TestTogglePicksTheRightDuplicate(t *testing.T) {
	got := apply(t, Toggle{Text: "tests", Checked: true, Nth: 1}, sameItems)
	if want := "## Backend\n- [ ] tests\n## Frontend\n- [x] tests\n"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestAddItem(t *testing.T) {
	cases := []struct{ body, want string }{
		{"- [x] a\n- [ ] b\n\nnotes\n", "- [x] a\n- [ ] b\n- [ ] new\n\nnotes\n"},
		{"- [ ] a\n  detail of a\n\nnotes\n", "- [ ] a\n  detail of a\n- [ ] new\n\nnotes\n"},
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
	now = func() time.Time { return time.Date(2026, 10, 3, 14, 2, 0, 0, time.Local) }
	t.Cleanup(func() { now = func() time.Time { return widget.Now() } })
	w := parseBody(body)
	w, res := w.Update("space") // the open item is first on the screen
	if want := (Toggle{Text: "Write tests", Checked: true, At: "2026-10-03 14:02"}); res.Op != want {
		t.Fatalf("Op = %+v, want %+v", res.Op, want)
	}
	if lines, cursor := draw(w, 40, true); !strings.HasPrefix(lines[cursor], "› ☑ Write tests") || !strings.HasSuffix(lines[cursor], "14:02") {
		t.Errorf("the item should look checked right away: %q", lines)
	}
}

func TestSpaceCarriesTheDuplicateIndex(t *testing.T) {
	now = func() time.Time { return time.Date(2026, 10, 3, 14, 2, 0, 0, time.Local) }
	t.Cleanup(func() { now = func() time.Time { return widget.Now() } })
	w := parseBody(sameItems)
	w, _ = w.Update("j")
	_, res := w.Update("space")
	if want := (Toggle{Text: "tests", Checked: true, Nth: 1, At: "2026-10-03 14:02"}); res.Op != want {
		t.Errorf("Op = %+v, want %+v", res.Op, want)
	}
}

func TestXIsNotAToggleKey(t *testing.T) {
	_, res := parseBody(body).Update("x")
	if res.Op != nil || Kind.Handles("x") {
		t.Error("x belongs to the screen (archive), not to the checklist")
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

func TestSyncClampsCursor(t *testing.T) {
	w := parseBody(body)
	w, _ = w.Update("j")
	w, _ = w.Update("j")
	w = w.Sync(doc.Document{Body: "- [ ] only\n"})
	if lines, cursor := draw(w, 40, true); cursor < 0 || lines[cursor] != "› ☐ only" {
		t.Errorf("Draw = %q", lines)
	}
}

func TestKind(t *testing.T) {
	if got := string(Kind.Template("Release")); got != "---\ntype: checklist\ntitle: Release\n---\n" {
		t.Errorf("Template = %q", got)
	}
	if Kind.Size(doc.Document{}) != widget.SizeHalf {
		t.Errorf("Size = %q", Kind.Size(doc.Document{}))
	}
	w := Kind.Parse(doc.Parse(Kind.Template("Release")))
	if lines, _ := draw(w, 50, false); strings.Contains(strings.Join(lines, "\n"), "0/1") {
		t.Errorf("a new checklist must not contain a blank item: %q", lines)
	}
}

func TestSummaryIsTheProgress(t *testing.T) {
	if got := parseBody(body).Summary(); got != "2/3" {
		t.Errorf("Summary = %q", got)
	}
	if got := parseBody("no items\n").Summary(); got != "" {
		t.Errorf("Summary without items = %q", got)
	}
}

func TestKindDescribesItself(t *testing.T) {
	if Kind.Icon == "" || widget.Width(Kind.Icon) != 1 || len(Kind.Hint) == 0 || len(Kind.Hint)%2 != 0 || Kind.Blurb == "" {
		t.Errorf("Kind needs a one-cell icon, a key hint and a blurb: %+v", Kind)
	}
	example := Kind.Parse(doc.Parse([]byte(Kind.Example)))
	if got := example.Summary(); got == "" {
		t.Errorf("the example should be a checklist with items, summary = %q", got)
	}
}
