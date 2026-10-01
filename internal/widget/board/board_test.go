package board

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"

	"github.com/LeeSwallow/stickypane/internal/doc"
	"github.com/LeeSwallow/stickypane/internal/widget"
)

const src = "---\ntype: board\ntitle: Auth\n---\n" + body

const body = "## To do\n- payments\n## Doing\n- login API\n  - refresh token later\n## Done\n- schema\n"

func parseSrc(s string) widget.Widget { return Kind.Parse(doc.Parse([]byte(s))) }

// draw returns the board without colors, plus the cursor line.
func draw(w widget.Widget, width int, active bool) ([]string, int) {
	out, cursor := w.Draw(width, active)
	return strings.Split(ansi.Strip(out), "\n"), cursor
}

func TestDrawShowsColumnsSideBySide(t *testing.T) {
	lines, _ := draw(parseSrc(src), 60, false)
	want := []string{
		"To do (1)           Doing (1)           Done (1)",
		"──────────────────  ──────────────────  ──────────────────",
		"▎ payments          ▎ login API         ▎ schema",
	}
	if strings.Join(lines, "\n") != strings.Join(want, "\n") {
		t.Errorf("Draw:\n%s\nwant:\n%s", strings.Join(lines, "\n"), strings.Join(want, "\n"))
	}
}

func TestDrawShowsEveryCardAndWrapsLongOnes(t *testing.T) {
	long := "프로젝트 개념 요약이 인용 하나의 revision 이 달라도 통째로 실패함"
	w := parseSrc("## A\n- 1\n- 2\n- 3\n- 4\n- 5\n- 6\n- " + long + "\n## B\n- x\n")
	lines, _ := draw(w, 44, false)
	text := strings.Join(lines, "\n")
	if strings.Contains(text, "more") || strings.Contains(text, "…") {
		t.Errorf("an open board must not fold or cut cards:\n%s", text)
	}
	for _, card := range []string{"1", "6", "x"} {
		if !strings.Contains(text, "▎ "+card) {
			t.Errorf("card %q is missing:\n%s", card, text)
		}
	}
	joined := strings.Join(strings.Fields(text), " ")
	for _, word := range []string{"프로젝트", "revision", "실패함"} {
		if !strings.Contains(joined, word) {
			t.Errorf("the long card lost %q when wrapping:\n%s", word, text)
		}
	}
	for _, line := range lines {
		if got := widget.Width(line); got > 44 {
			t.Errorf("line %q is %d cells wide, want at most 44", line, got)
		}
	}
}

func TestDrawPutsABlankLineBetweenCards(t *testing.T) {
	lines, _ := draw(parseSrc("## A\n- one\n- two\n"), 40, false)
	if want := []string{"A (2)", strings.Repeat("─", 38), "▎ one", "", "▎ two"}; strings.Join(lines, "|") != strings.Join(want, "|") {
		t.Errorf("Draw = %q, want %q", lines, want)
	}
}

func TestDrawStacksColumnsWhenNarrow(t *testing.T) {
	lines, _ := draw(parseSrc(src), 30, false)
	rule := strings.Repeat("─", 29)
	want := []string{"To do (1)", rule, "▎ payments", "", "Doing (1)", rule, "▎ login API", "", "Done (1)", rule, "▎ schema"}
	if strings.Join(lines, "|") != strings.Join(want, "|") {
		t.Errorf("Draw = %q\nwant  %q", lines, want)
	}
}

func TestCursorShowsOnlyWhenActive(t *testing.T) {
	w := parseSrc(src)
	if lines, cursor := draw(w, 60, false); cursor != -1 || strings.Contains(strings.Join(lines, "\n"), "›") {
		t.Errorf("an inactive board must not show a cursor (cursor = %d):\n%s", cursor, strings.Join(lines, "\n"))
	}
	lines, cursor := draw(w, 60, true)
	if cursor < 0 || !strings.Contains(lines[cursor], "› payments") {
		t.Errorf("cursor = %d, line = %q; want the line of the selected card", cursor, lines[min(max(cursor, 0), len(lines)-1)])
	}
}

func TestCursorLineFollowsTheSelection(t *testing.T) {
	w := parseSrc("intro line\n\n## A\n- one\n- two\n- three\n## B\n- x\n")
	w, _ = w.Update("j")
	w, _ = w.Update("j")
	lines, cursor := draw(w, 60, true)
	if cursor < 0 || !strings.Contains(lines[cursor], "› three") {
		t.Fatalf("cursor = %d, lines = %q", cursor, lines)
	}
	lines, cursor = draw(w, 20, true) // stacked
	if cursor < 0 || !strings.Contains(lines[cursor], "› three") {
		t.Errorf("stacked: cursor = %d, lines = %q", cursor, lines)
	}
}

func TestDetailShowsUnderTheSelectedCard(t *testing.T) {
	w := parseSrc(src)
	lines, _ := draw(w, 60, true)
	if strings.Contains(strings.Join(lines, "\n"), "refresh token") {
		t.Fatal("details belong to the selected card only")
	}
	w, _ = w.Update("l")
	lines, cursor := draw(w, 60, true)
	if !strings.Contains(lines[cursor], "› login API") || !strings.HasPrefix(strings.TrimLeft(lines[cursor+1][20:], " "), "refresh token") {
		t.Errorf("the detail should sit right under the selected card:\n%s", strings.Join(lines, "\n"))
	}
}

func TestDrawShowsTheDescription(t *testing.T) {
	lines, _ := draw(parseSrc("Sprint goal: ship login\nsecond line\n\n## A\n- x\n"), 60, false)
	if lines[0] != "Sprint goal: ship login" || lines[1] != "second line" || lines[2] != "" {
		t.Errorf("the description should come first, followed by a blank line: %q", lines)
	}
}

func TestBoardWithoutColumnsStillShowsItsText(t *testing.T) {
	lines, cursor := draw(parseSrc("just text\nmore text\n"), 60, true)
	text := strings.Join(lines, "\n")
	for _, want := range []string{`"## Name"`, "just text", "more text"} {
		if !strings.Contains(text, want) {
			t.Errorf("Draw should contain %q:\n%s", want, text)
		}
	}
	if cursor != -1 {
		t.Errorf("cursor = %d, want -1 without columns", cursor)
	}
}

func TestLinesThatAreNotListItemsAreStillCards(t *testing.T) {
	lines, _ := draw(parseSrc("## To do\nask Bob about the API\n1. first step\n- normal card\n## Done\n"), 70, false)
	text := strings.Join(lines, "\n")
	for _, want := range []string{"To do (3)", "ask Bob about the API", "1. first step", "normal card"} {
		if !strings.Contains(text, want) {
			t.Errorf("Draw should contain %q:\n%s", want, text)
		}
	}
}

func TestDrawKeepsWidthWithWideText(t *testing.T) {
	w := parseSrc("설명 줄이 아주 길어서 줄을 바꿔야 하는 경우입니다 정말로요\n## 할 일\n- 결제 연동을 다음 주까지 마무리하기\n  - 상세 설명도 길게 적어서 줄이 바뀌어야 합니다\n## 진행 중\n- 로그인 API 📌\n## 완료\n- DB 스키마\n")
	for _, width := range []int{1, 8, 20, 34, 60} {
		for _, active := range []bool{false, true} {
			out, _ := w.Draw(width, active)
			for _, line := range strings.Split(out, "\n") {
				if got := widget.Width(line); got > width {
					t.Errorf("width %d: line %q is %d cells wide", width, ansi.Strip(line), got)
				}
			}
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
	lines, cursor := draw(w.Sync(d), 60, true)
	if cursor < 0 || !strings.Contains(lines[cursor], "› login API") {
		t.Errorf("cursor should follow the moved card:\n%s", strings.Join(lines, "\n"))
	}
}

func TestMoveKeyAtEdgeDoesNothing(t *testing.T) {
	_, res := parseSrc(src).Update("H")
	if res.Op != nil {
		t.Errorf("H in the first column should do nothing, got %+v", res.Op)
	}
}

func TestReorderKeys(t *testing.T) {
	_, res := parseSrc("## A\n- one\n- two\n").Update("J")
	if want := (ReorderCard{Col: "A", Text: "one", Delta: 1}); res.Op != want {
		t.Errorf("J: Op = %+v, want %+v", res.Op, want)
	}
	_, res = parseSrc("## A\n- one\n- two\n").Update("K")
	if res.Op != nil {
		t.Errorf("K on the first card should do nothing, got %+v", res.Op)
	}
}

func TestKeysCarryTheDuplicateIndex(t *testing.T) {
	const duplicates = "## A\n- same\n  first detail\n- same\n  second detail\n- z\n## B\n"
	w := parseSrc(duplicates)
	w, _ = w.Update("j")
	_, res := w.Update("L")
	if want := (MoveCard{From: "A", To: "B", Text: "same", Nth: 1}); res.Op != want {
		t.Errorf("L: Op = %+v, want %+v", res.Op, want)
	}
	w = parseSrc(duplicates)
	w, _ = w.Update("j")
	_, res = w.Update("J")
	if want := (ReorderCard{Col: "A", Text: "same", Nth: 1, Delta: 1}); res.Op != want {
		t.Errorf("J: Op = %+v, want %+v", res.Op, want)
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

func TestEmptyBoardIgnoresKeys(t *testing.T) {
	w := parseSrc("no columns here\n")
	for _, key := range strings.Fields(Kind.Keys) {
		var res widget.Result
		w, res = w.Update(key)
		if res.Op != nil || res.Prompt != nil {
			t.Errorf("%s on an empty board returned %+v", key, res)
		}
	}
	w.Draw(40, true)
}

func TestSyncClampsCursor(t *testing.T) {
	w := parseSrc(src)
	w, _ = w.Update("l")
	w, _ = w.Update("l")
	w = w.Sync(doc.Parse([]byte("## Only\n- card\n")))
	if lines, cursor := draw(w, 40, true); cursor < 0 || !strings.Contains(lines[cursor], "› card") {
		t.Errorf("cursor should clamp to the remaining column: %q", lines)
	}
}

func TestSummaryCountsCards(t *testing.T) {
	if got := parseSrc(src).Summary(); got != "3 cards" {
		t.Errorf("Summary = %q", got)
	}
	if got := parseSrc("## A\n- one\n").Summary(); got != "1 card" {
		t.Errorf("Summary = %q", got)
	}
	if got := parseSrc("no columns\n").Summary(); got != "" {
		t.Errorf("Summary without columns = %q", got)
	}
}

func TestKindDescribesItself(t *testing.T) {
	if Kind.Icon == "" || widget.Width(Kind.Icon) != 1 || len(Kind.Hint) == 0 || len(Kind.Hint)%2 != 0 || Kind.Blurb == "" {
		t.Errorf("Kind needs a one-cell icon, a key hint and a blurb: %+v", Kind)
	}
	example := Kind.Parse(doc.Parse([]byte(Kind.Example)))
	if out, _ := example.Draw(40, false); !strings.Contains(ansi.Strip(out), "(") {
		t.Errorf("the example should draw as a board with columns:\n%s", out)
	}
}

func TestKind(t *testing.T) {
	got := string(Kind.Template("Auth"))
	want := "---\ntype: board\ntitle: Auth\n---\n## To do\n\n## Doing\n\n## Done\n"
	if got != want {
		t.Errorf("Template = %q", got)
	}
	if Kind.Name != "board" || Kind.Size(doc.Document{}) != widget.SizePage {
		t.Errorf("Kind = %+v", Kind)
	}
	for _, key := range []string{"h", "l", "j", "k", "H", "L", "J", "K", "n", "left", "right", "up", "down"} {
		if !Kind.Handles(key) {
			t.Errorf("the board should handle %q", key)
		}
	}
}
