package board

import (
	"errors"
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"

	"github.com/LeeSwallow/stickypane/internal/doc"
	"github.com/LeeSwallow/stickypane/internal/widget"
)

const src = "---\ntype: board\ntitle: Auth\n---\n" + body

const body = "## To do\n- payments\n## Doing\n- login API\n  - refresh token later\n## Done\n- schema\n"

func parseSrc(s string) widget.Widget { return Kind.Parse(doc.Parse([]byte(s))) }

func apply(t *testing.T, op doc.Op, body string) string {
	t.Helper()
	d, err := op.Apply(doc.Document{Body: body})
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}
	return d.Body
}

func TestPreviewShowsColumnsSideBySide(t *testing.T) {
	got := ansi.Strip(parseSrc(src).Preview(60))
	want := "To do (1)           Doing (1)           Done (1)\n" +
		"payments            login API           schema"
	if got != want {
		t.Errorf("Preview:\n%s\nwant:\n%s", got, want)
	}
}

func TestPreviewStacksColumnsWhenNarrow(t *testing.T) {
	got := ansi.Strip(parseSrc(src).Preview(30))
	want := "To do (1)\n  payments\nDoing (1)\n  login API\nDone (1)\n  schema"
	if got != want {
		t.Errorf("Preview:\n%s\nwant:\n%s", got, want)
	}
}

func TestPreviewLimitsCardsPerColumn(t *testing.T) {
	got := ansi.Strip(parseSrc("## A\n- 1\n- 2\n- 3\n- 4\n- 5\n- 6\n- 7\n").Preview(40))
	if want := "A (7)\n1\n2\n3\n4\n5\n+2 more"; got != want {
		t.Errorf("Preview = %q, want %q", got, want)
	}
}

func TestPreviewWithoutColumnsExplainsFormat(t *testing.T) {
	got := ansi.Strip(parseSrc("just text\n").Preview(60))
	if !strings.Contains(got, `"## Name"`) {
		t.Errorf("Preview = %q, want a hint about headings", got)
	}
}

func TestPreviewKeepsWidthWithWideText(t *testing.T) {
	w := parseSrc("## 할 일\n- 결제 연동을 다음 주까지 마무리하기\n## 진행 중\n- 로그인 API 📌\n## 완료\n- DB 스키마\n")
	for _, width := range []int{20, 34, 60} {
		for _, line := range strings.Split(w.Preview(width), "\n") {
			if got := widget.Width(line); got > width {
				t.Errorf("width %d: line %q is %d cells wide", width, ansi.Strip(line), got)
			}
		}
	}
}

func TestMoveCard(t *testing.T) {
	got := apply(t, MoveCard{From: "Doing", To: "Done", Text: "login API"}, body)
	want := "## To do\n- payments\n## Doing\n## Done\n- schema\n- login API\n  - refresh token later\n"
	if got != want {
		t.Errorf("got %q\nwant %q", got, want)
	}
}

func TestMoveCardIntoEmptyColumn(t *testing.T) {
	got := apply(t, MoveCard{From: "A", To: "B", Text: "x"}, "## A\n- x\n\n## B\n\n## C\n")
	if want := "## A\n\n## B\n- x\n\n## C\n"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestMoveCardKeepsCRLF(t *testing.T) {
	got := apply(t, MoveCard{From: "A", To: "B", Text: "x"}, "## A\r\n- x\r\n## B\r\n")
	if want := "## A\r\n## B\r\n- x\r\n"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestMoveCardConflicts(t *testing.T) {
	for _, op := range []doc.Op{
		MoveCard{From: "Doing", To: "Done", Text: "gone"},
		MoveCard{From: "Nowhere", To: "Done", Text: "login API"},
		MoveCard{From: "Doing", To: "Nowhere", Text: "login API"},
		ReorderCard{Col: "Doing", Text: "gone", Delta: 1},
		AddCard{Col: "Nowhere", Text: "x"},
	} {
		d, err := op.Apply(doc.Document{Body: body})
		if !errors.Is(err, doc.ErrConflict) {
			t.Errorf("%+v: err = %v, want ErrConflict", op, err)
		}
		if d.Body != body {
			t.Errorf("%+v: body changed on conflict", op)
		}
	}
}

func TestReorderCard(t *testing.T) {
	list := "## A\n- one\n  - detail\n- two\n- three\n"
	cases := []struct {
		text  string
		delta int
		want  string
	}{
		{"one", 1, "## A\n- two\n- one\n  - detail\n- three\n"},
		{"three", -1, "## A\n- one\n  - detail\n- three\n- two\n"},
		{"one", -1, list},
		{"three", 1, list},
	}
	for _, c := range cases {
		if got := apply(t, ReorderCard{Col: "A", Text: c.text, Delta: c.delta}, list); got != c.want {
			t.Errorf("reorder %q by %d: got %q, want %q", c.text, c.delta, got, c.want)
		}
	}
}

func TestAddCard(t *testing.T) {
	cases := []struct{ body, col, want string }{
		{"## A\n\n## B\n", "A", "## A\n- new\n\n## B\n"},
		{"## A\n- one\n  - detail\n## B\n", "A", "## A\n- one\n  - detail\n- new\n## B\n"},
		{"## A", "A", "## A\n- new"},
		{"## A\r\n- one\r\n", "A", "## A\r\n- one\r\n- new\r\n"},
	}
	for _, c := range cases {
		if got := apply(t, AddCard{Col: c.col, Text: "new"}, c.body); got != c.want {
			t.Errorf("add to %q: got %q, want %q", c.body, got, c.want)
		}
	}
}

func TestMoveKeyEmitsOpAndFollowsCard(t *testing.T) {
	w := parseSrc(src)
	w, res := w.Update("l")
	if res.Op != nil {
		t.Fatalf("l should only move the cursor, got %+v", res.Op)
	}
	w, res = w.Update("L")
	want := MoveCard{From: "Doing", To: "Done", Text: "login API"}
	if res.Op != want {
		t.Fatalf("Op = %+v, want %+v", res.Op, want)
	}
	d, err := res.Op.Apply(doc.Parse([]byte(src)))
	if err != nil {
		t.Fatal(err)
	}
	view := ansi.Strip(w.Sync(d).View(60, 10))
	if !strings.Contains(view, "› login API") {
		t.Errorf("cursor should follow the moved card:\n%s", view)
	}
}

func TestMoveKeyAtEdgeDoesNothing(t *testing.T) {
	_, res := parseSrc(src).Update("H")
	if res.Op != nil {
		t.Errorf("H in the first column should do nothing, got %+v", res.Op)
	}
}

func TestReorderKeys(t *testing.T) {
	w := parseSrc("## A\n- one\n- two\n")
	_, res := w.Update("J")
	if want := (ReorderCard{Col: "A", Text: "one", Delta: 1}); res.Op != want {
		t.Errorf("J: Op = %+v, want %+v", res.Op, want)
	}
	_, res = parseSrc("## A\n- one\n- two\n").Update("K")
	if res.Op != nil {
		t.Errorf("K on the first card should do nothing, got %+v", res.Op)
	}
}

func TestNewCardPrompt(t *testing.T) {
	_, res := parseSrc(src).Update("n")
	if res.Prompt == nil {
		t.Fatal("n should ask for the card text")
	}
	if res.Prompt.Label != "New card in To do" {
		t.Errorf("Label = %q", res.Prompt.Label)
	}
	if got, want := res.Prompt.Submit("write tests"), (AddCard{Col: "To do", Text: "write tests"}); got != want {
		t.Errorf("Submit = %+v, want %+v", got, want)
	}
}

func TestViewShowsDetailOfSelectedCard(t *testing.T) {
	w := parseSrc(src)
	w, _ = w.Update("l")
	view := ansi.Strip(w.View(60, 10))
	for _, want := range []string{"› login API", "refresh token later", "  payments"} {
		if !strings.Contains(view, want) {
			t.Errorf("View should contain %q:\n%s", want, view)
		}
	}
	if n := strings.Count(view, "\n") + 1; n > 10 {
		t.Errorf("View is %d lines tall, want at most 10", n)
	}
}

func TestViewScrollsColumnsWhenNarrow(t *testing.T) {
	w := parseSrc(src)
	w, _ = w.Update("l")
	w, _ = w.Update("l")
	view := ansi.Strip(w.View(20, 6))
	if !strings.Contains(view, "› schema") {
		t.Errorf("the selected column must stay visible:\n%s", view)
	}
	for _, line := range strings.Split(view, "\n") {
		if widget.Width(line) > 20 {
			t.Errorf("line %q is wider than 20 cells", line)
		}
	}
}

func TestEmptyBoardIgnoresKeys(t *testing.T) {
	w := parseSrc("no columns here\n")
	for _, key := range []string{"h", "l", "j", "k", "H", "L", "J", "K", "n"} {
		var res widget.Result
		w, res = w.Update(key)
		if res.Op != nil || res.Prompt != nil {
			t.Errorf("%s on an empty board returned %+v", key, res)
		}
	}
	w.View(40, 5)
}

func TestSyncClampsCursor(t *testing.T) {
	w := parseSrc(src)
	w, _ = w.Update("l")
	w, _ = w.Update("l")
	w = w.Sync(doc.Parse([]byte("## Only\n- card\n")))
	if view := ansi.Strip(w.View(40, 6)); !strings.Contains(view, "› card") {
		t.Errorf("cursor should clamp to the remaining column:\n%s", view)
	}
}

func TestKindTemplate(t *testing.T) {
	got := string(Kind.Template("Auth"))
	want := "---\ntype: board\ntitle: Auth\n---\n## To do\n\n## Doing\n\n## Done\n"
	if got != want {
		t.Errorf("Template = %q", got)
	}
	if !Kind.FullRow || Kind.Name != "board" {
		t.Errorf("Kind = %+v", Kind)
	}
}
