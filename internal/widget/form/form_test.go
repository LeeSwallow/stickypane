package form

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/x/ansi"

	"github.com/LeeSwallow/stickypane/internal/doc"
	"github.com/LeeSwallow/stickypane/internal/widget"
	"github.com/LeeSwallow/stickypane/internal/widget/note"
)

const deploy = `---
type: form
title: 배포
---
배포 전에 확인합니다.

## 어디에 배포할까요?
- ( ) staging
  먼저 올려 보고 확인합니다.
- ( ) production

## 같이 할 일
- [ ] 마이그레이션
- [x] 캐시 비우기

## 메모
> 

[ 제출 ] [ 취소 ]
`

var kind = NewKind(note.Plain)

func parseSrc(s string) widget.Widget { return kind.Parse(doc.Parse([]byte(s))) }

func draw(t *testing.T, w widget.Widget, width int) ([]string, widget.Span) {
	t.Helper()
	out, at := w.Draw(width, true)
	for _, line := range strings.Split(out, "\n") {
		if got := widget.Width(line); got > width {
			t.Errorf("line %q is %d cells wide, want at most %d", ansi.Strip(line), got, width)
		}
	}
	return strings.Split(ansi.Strip(out), "\n"), at
}

// press sends keys and applies every Op to the file, as the app does.
func press(t *testing.T, src string, w widget.Widget, keys ...string) (string, widget.Widget, widget.Result) {
	t.Helper()
	var res widget.Result
	for _, k := range keys {
		w, res = w.Update(k)
		if res.Op != nil {
			d, err := res.Op.Apply(doc.Parse([]byte(src)))
			if err != nil {
				t.Fatalf("key %q: %v", k, err)
			}
			src = string(d.Bytes())
			w = w.Sync(doc.Parse([]byte(src)))
		}
	}
	return src, w, res
}

func TestKind(t *testing.T) {
	if kind.Name != "form" || widget.Width(kind.Icon) != 1 || kind.Blurb == "" || kind.Example == "" {
		t.Errorf("Kind = %+v", kind)
	}
	for _, k := range []string{"j", "k", "up", "down", "space", "enter"} {
		if !kind.Handles(k) {
			t.Errorf("a form should take %q", k)
		}
	}
	tmpl := string(kind.Template("Deploy"))
	if !strings.HasPrefix(tmpl, "---\ntype: form\ntitle: Deploy\n---\n") || !strings.Contains(tmpl, "- ( ) ") || !strings.Contains(tmpl, "[ Submit ]") {
		t.Errorf("a new form should start with something to answer: %q", tmpl)
	}
}

func TestDrawShowsTheDocumentWithItsControls(t *testing.T) {
	lines, at := draw(t, parseSrc(deploy), 44)
	text := strings.Join(lines, "\n")
	for _, want := range []string{
		"배포 전에 확인합니다.", "## 어디에 배포할까요?",
		"› ○ staging", "    먼저 올려 보고 확인합니다.", "  ○ production",
		"  ☐ 마이그레이션", "  ☑ 캐시 비우기",
		"│ 제출 │ │ 취소 │",
	} {
		if !strings.Contains(text, want) {
			t.Errorf("Draw should contain %q:\n%s", want, text)
		}
	}
	for _, gone := range []string{"- ( )", "- [ ]", "[ 제출 ]", "> "} {
		if strings.Contains(text, gone) {
			t.Errorf("the source %q should be drawn as a control:\n%s", gone, text)
		}
	}
	if !at.Ok() || !strings.Contains(lines[at.Start], "staging") || at.End != at.Start+2 {
		t.Errorf("the selection should cover the option and its detail: %+v\n%s", at, text)
	}
}

func TestSpacingFollowsTheFileNotTheRenderer(t *testing.T) {
	// A renderer such as Glamour pads its output with blank lines.
	padded := NewKind(func(md string, width int) string { return "\n   \n" + note.Plain(md, width) + "\n   \n\n" })
	w := padded.Parse(doc.Parse([]byte("intro\n\n## Q\n- ( ) a\n- ( ) b\n\n\n[ Go ]\n\n")))
	lines, _ := draw(t, w, 30)
	want := "intro\n\n## Q\n› ○ a\n  ○ b\n\n╭────╮\n│ Go │\n╰────╯"
	if got := strings.Join(lines, "\n"); got != want {
		t.Errorf("Draw =\n%s\nwant\n%s", got, want)
	}
}

func TestChoosingOneOptionClearsTheOthersInItsGroup(t *testing.T) {
	src, w, _ := press(t, deploy, parseSrc(deploy), "space")
	if !strings.Contains(src, "- (x) staging") {
		t.Fatalf("space should choose the option:\n%s", src)
	}
	src, w, _ = press(t, src, w, "j", "enter")
	if !strings.Contains(src, "- ( ) staging") || !strings.Contains(src, "- (x) production") {
		t.Errorf("choosing another option should clear the first:\n%s", src)
	}
	if !strings.Contains(src, "- [x] 캐시 비우기") || !strings.Contains(src, "  먼저 올려 보고 확인합니다.") {
		t.Errorf("other lines must stay as they are:\n%s", src)
	}
	lines, _ := draw(t, w, 44)
	if !strings.Contains(strings.Join(lines, "\n"), "› ◉ production") {
		t.Errorf("the chosen option should be marked:\n%s", strings.Join(lines, "\n"))
	}
	src, _, _ = press(t, src, w, "space")
	if !strings.Contains(src, "- ( ) production") {
		t.Errorf("choosing a chosen option takes the choice back:\n%s", src)
	}
}

func TestCheckboxesAreIndependent(t *testing.T) {
	src, _, _ := press(t, deploy, parseSrc(deploy), "j", "j", "space", "j", "space")
	if !strings.Contains(src, "- [x] 마이그레이션") || !strings.Contains(src, "- [ ] 캐시 비우기") {
		t.Errorf("each box toggles on its own:\n%s", src)
	}
}

func TestATextFieldIsFilledThroughAPrompt(t *testing.T) {
	w := parseSrc(deploy)
	_, w, res := press(t, deploy, w, "j", "j", "j", "j", "enter")
	if res.Prompt == nil || res.Prompt.Label != "메모" || res.Prompt.Initial != "" || !res.Prompt.Empty {
		t.Fatalf("enter on a field should ask for its text: %+v", res.Prompt)
	}
	d, err := res.Prompt.Submit("금요일 전에").Apply(doc.Parse([]byte(deploy)))
	if err != nil {
		t.Fatal(err)
	}
	src := string(d.Bytes())
	if !strings.Contains(src, "## 메모\n> 금요일 전에\n") {
		t.Fatalf("the answer should be written on the field's line:\n%s", src)
	}
	w = w.Sync(d)
	lines, _ := draw(t, w, 44)
	if !strings.Contains(strings.Join(lines, "\n"), "┃ 금요일 전에") {
		t.Errorf("the field should show its text in a box:\n%s", strings.Join(lines, "\n"))
	}
	_, res2 := w.Update("enter")
	if res2.Prompt == nil || res2.Prompt.Initial != "금요일 전에" {
		t.Errorf("editing a field should start from its text: %+v", res2.Prompt)
	}
	// The agent rewrote the field in the meantime: the old intent is stale.
	if _, err := res.Prompt.Submit("다른 답").Apply(d); !errors.Is(err, doc.ErrConflict) {
		t.Errorf("filling a field that changed should be a conflict, got %v", err)
	}
}

func TestPressingAButtonSubmits(t *testing.T) {
	now = func() time.Time { return time.Date(2026, 10, 2, 14, 3, 5, 0, time.UTC) }
	defer func() { now = time.Now }()

	w := parseSrc(deploy)
	if got := w.Summary(); got != "1/3" {
		t.Errorf("Summary = %q, want the answered questions", got)
	}
	src, w, _ := press(t, deploy, w, "j", "j", "j", "j", "j", "enter")
	d := doc.Parse([]byte(src))
	if v, _ := d.Get("submitted"); v != "제출" {
		t.Fatalf("the pressed button should be recorded:\n%s", src)
	}
	if v, _ := d.Get("submitted_at"); v != "2026-10-02T14:03:05Z" {
		t.Errorf("submitted_at = %q", v)
	}
	if got := w.Summary(); got != "✓ 제출" {
		t.Errorf("Summary = %q", got)
	}
	lines, _ := draw(t, w, 44)
	if text := strings.Join(lines, "\n"); !strings.Contains(text, "✓ 제출") {
		t.Errorf("the pressed button should be marked:\n%s", text)
	}
	// l moves along the buttons too.
	src, _, _ = press(t, src, w, "l", "space")
	if v, _ := doc.Parse([]byte(src)).Get("submitted"); v != "취소" {
		t.Errorf("the second button should replace the first:\n%s", src)
	}
}

func TestChangingAnAnswerTakesTheSubmissionBack(t *testing.T) {
	src := strings.Replace(deploy, "title: 배포\n", "title: 배포\nsubmitted: 제출\nsubmitted_at: 2026-10-02T14:03:05Z\n", 1)
	src, _, _ = press(t, src, parseSrc(src), "space")
	d := doc.Parse([]byte(src))
	if _, ok := d.Get("submitted"); ok {
		t.Errorf("answers changed after submitting: the form is no longer submitted:\n%s", src)
	}
	if _, ok := d.Get("submitted_at"); ok {
		t.Errorf("submitted_at should go too:\n%s", src)
	}
	if v, _ := d.Get("title"); v != "배포" {
		t.Errorf("other keys must stay:\n%s", src)
	}
}

func TestAFormWithoutButtonsGetsASubmitButton(t *testing.T) {
	src := "---\ntype: form\n---\n계속할까요?\n- ( ) 네\n- ( ) 아니요\n"
	w := parseSrc(src)
	lines, _ := draw(t, w, 30)
	if !strings.Contains(strings.Join(lines, "\n"), "│ Submit │") {
		t.Fatalf("a form can always be submitted:\n%s", strings.Join(lines, "\n"))
	}
	out, _, _ := press(t, src, w, "j", "j", "enter")
	if v, _ := doc.Parse([]byte(out)).Get("submitted"); v != "Submit" {
		t.Errorf("the built-in button should submit:\n%s", out)
	}
	if !strings.HasSuffix(out, "- ( ) 아니요\n") {
		t.Errorf("the built-in button is not written into the body:\n%s", out)
	}
}

func TestReadGivesTheAnswers(t *testing.T) {
	src := strings.Replace(deploy, "- ( ) production", "- (x) production", 1)
	src = strings.Replace(src, "> \n", "> 금요일 전에\n", 1)
	a := Read(doc.Parse([]byte(src)))
	if a.Submitted || a.Button != "" {
		t.Errorf("not submitted yet: %+v", a)
	}
	want := "submitted: no\n어디에 배포할까요?: production\n같이 할 일: 캐시 비우기\n메모: 금요일 전에\n"
	if got := a.String(); got != want {
		t.Errorf("String =\n%s\nwant\n%s", got, want)
	}

	src = strings.Replace(src, "title: 배포\n", "title: 배포\nsubmitted: 제출\nsubmitted_at: 2026-10-02T14:03:05Z\n", 1)
	src = strings.Replace(src, "- [ ] 마이그레이션", "- [x] 마이그레이션", 1)
	a = Read(doc.Parse([]byte(src)))
	if !a.Submitted || a.Button != "제출" || a.At != "2026-10-02T14:03:05Z" {
		t.Errorf("Read = %+v", a)
	}
	if len(a.Answers) != 3 || strings.Join(a.Answers[1].Values, "|") != "마이그레이션|캐시 비우기" {
		t.Errorf("Answers = %+v", a.Answers)
	}
	if !strings.HasPrefix(a.String(), "submitted: 제출\nat: 2026-10-02T14:03:05Z\n") {
		t.Errorf("String = %q", a.String())
	}
}

func TestQuestionsWithoutAHeadingAreNamedByTheLineAbove(t *testing.T) {
	a := Read(doc.Parse([]byte("계속할까요?\n- (x) 네\n- ( ) 아니요\n\n- [ ] 혼자 있는 상자\n\n> \n")))
	var names []string
	for _, q := range a.Answers {
		names = append(names, q.Question)
	}
	if got := strings.Join(names, "|"); got != "계속할까요?|question 2|question 3" {
		t.Errorf("questions = %q", got)
	}
	if got := a.Answers[2].Values; len(got) != 0 {
		t.Errorf("an empty field has no answer: %q", got)
	}
}

func TestABlankLineEndsAQuestion(t *testing.T) {
	src := "- ( ) red\n- (x) blue\n\n- ( ) small\n- ( ) large\n"
	out, _, _ := press(t, src, parseSrc(src), "j", "j", "j", "space")
	if out != "- ( ) red\n- (x) blue\n\n- ( ) small\n- (x) large\n" {
		t.Errorf("options after a blank line are another question:\n%s", out)
	}
	if a := Read(doc.Parse([]byte(out))); len(a.Answers) != 2 {
		t.Errorf("Answers = %+v", a.Answers)
	}
}

func TestIndentedControlsAreControls(t *testing.T) {
	src := "## Where?\n  - ( ) staging\n  - ( ) production\n  [ Deploy ] [ Cancel ]\n"
	a := Read(doc.Parse([]byte(src)))
	if len(a.Answers) != 1 {
		t.Fatalf("options indented like a nested list are still options: %+v", a.Answers)
	}
	lines, _ := draw(t, parseSrc(src), 40)
	if text := strings.Join(lines, "\n"); !strings.Contains(text, "│ Deploy │") || strings.Contains(text, "Submit") {
		t.Errorf("a button line under an option is still a button line:\n%s", text)
	}
	out, _, _ := press(t, src, parseSrc(src), "j", "space")
	if !strings.Contains(out, "  - (x) production\n") {
		t.Errorf("the indentation must survive a choice:\n%s", out)
	}
}

func TestLinesThatOnlyLookLikeControls(t *testing.T) {
	src := "- [x](http://example.com/x) a link\n[   ]\n- ( ) real\n"
	f := parseSrc(src)
	if a := Read(doc.Parse([]byte(src))); len(a.Answers) != 1 || len(a.Answers[0].Values) != 0 {
		t.Errorf("a link is not a ticked box: %+v", a.Answers)
	}
	lines, _ := draw(t, f, 40)
	if text := strings.Join(lines, "\n"); !strings.Contains(text, "│ Submit │") || strings.Contains(text, "✓") {
		t.Errorf("a button without a name is not a button:\n%s", text)
	}
}

func TestFillingAFieldWithTheSameTextKeepsTheSubmission(t *testing.T) {
	src := "---\ntype: form\nsubmitted: Go\n---\n> same\n\n[ Go ]\n"
	d, err := (Fill{Nth: 0, Old: "same", Text: "same"}).Apply(doc.Parse([]byte(src)))
	if err != nil || string(d.Bytes()) != src {
		t.Errorf("nothing changed, so nothing should be written: %v\n%s", err, d.Bytes())
	}
}

func TestTextFromTheFileIsCleaned(t *testing.T) {
	src := "Hello \x1b[2J\x1b]0;pwned\x07 world\n\n## Name \x1b[2J\n> \n"
	f := parseSrc(src)
	out, _ := f.Draw(60, true)
	if strings.Contains(out, "\x1b[2J") || strings.Contains(out, "\x07") || strings.Contains(out, "pwned\x07") {
		t.Errorf("escape sequences from the file must not reach the screen: %q", out)
	}
	_, res := f.Update("enter")
	if res.Prompt == nil || strings.ContainsAny(res.Prompt.Label, "\x1b\x07") {
		t.Errorf("the prompt label must be clean: %+v", res.Prompt)
	}
}

func TestLongerFencesHideTheirControls(t *testing.T) {
	src := "````markdown\n```\n- ( ) example\n```\n- ( ) still an example\n````\n- ( ) real\n"
	if a := Read(doc.Parse([]byte(src))); len(a.Answers) != 1 {
		t.Errorf("only the option after the long fence is real: %+v", a.Answers)
	}
}

func TestCodeBlocksAreNotControls(t *testing.T) {
	src := "예시:\n\n```\n- ( ) not an option\n[ Not a button ]\n> not a field\n```\n\n- ( ) 진짜 선택지\n"
	a := Read(doc.Parse([]byte(src)))
	if len(a.Answers) != 1 {
		t.Fatalf("only the option outside the code block is a question: %+v", a.Answers)
	}
	lines, _ := draw(t, parseSrc(src), 40)
	if text := strings.Join(lines, "\n"); !strings.Contains(text, "- ( ) not an option") || !strings.Contains(text, "[ Not a button ]") {
		t.Errorf("a code block is shown as written:\n%s", text)
	}
}

func TestOptionsWithTheSameTextAreToldApart(t *testing.T) {
	src := "## A\n- ( ) 네\n## B\n- ( ) 네\n"
	out, _, _ := press(t, src, parseSrc(src), "j", "space")
	if out != "## A\n- ( ) 네\n## B\n- (x) 네\n" {
		t.Errorf("the second 네 should be chosen:\n%s", out)
	}
}

func TestAStaleChoiceIsAConflict(t *testing.T) {
	_, res := parseSrc(deploy).Update("space")
	changed := strings.Replace(deploy, "- ( ) staging", "- ( ) 스테이징", 1)
	if _, err := res.Op.Apply(doc.Parse([]byte(changed))); !errors.Is(err, doc.ErrConflict) {
		t.Errorf("an option that is gone should be a conflict, got %v", err)
	}
	if _, err := (Submit{Label: "없는 버튼"}).Apply(doc.Parse([]byte(deploy))); !errors.Is(err, doc.ErrConflict) {
		t.Errorf("a button that is gone should be a conflict, got %v", err)
	}
}

func TestTheCursorStopsAtBothEnds(t *testing.T) {
	w := parseSrc(deploy)
	_, w, _ = press(t, deploy, w, "k", "k")
	lines, at := draw(t, w, 44)
	if !strings.Contains(lines[at.Start], "staging") {
		t.Errorf("k at the top stays on the first control: %q", lines[at.Start])
	}
	_, w, _ = press(t, deploy, w, "j", "j", "j", "j", "j", "j", "j", "j", "j")
	lines, at = draw(t, w, 44)
	if !at.Ok() || !strings.Contains(strings.Join(lines[at.Start:at.End], "\n"), "취소") {
		t.Errorf("j at the end stays on the last button: %+v", at)
	}
}

func TestNarrowAndEmptyFormsDoNotBreak(t *testing.T) {
	for _, width := range []int{1, 3, 8, 15, 30} {
		w := parseSrc(deploy)
		for i := 0; i < 8; i++ {
			draw(t, w, width)
			w, _ = w.Update("j")
		}
	}
	w := parseSrc("---\ntype: form\n---\n")
	lines, _ := draw(t, w, 30)
	if !strings.Contains(strings.Join(lines, "\n"), "Submit") {
		t.Errorf("an empty form still has its button: %q", lines)
	}
	if got := w.Summary(); got != "" {
		t.Errorf("Summary of an empty form = %q", got)
	}
}

func TestButtonsWrapWhenTheyDoNotFit(t *testing.T) {
	src := "[ 승인하고 배포 ] [ 다시 검토 요청 ] [ 취소 ]\n"
	lines, _ := draw(t, parseSrc(src), 24)
	text := strings.Join(lines, "\n")
	for _, want := range []string{"승인하고 배포", "다시 검토 요청", "취소"} {
		if !strings.Contains(text, want) {
			t.Errorf("every button must stay visible, missing %q:\n%s", want, text)
		}
	}
}
