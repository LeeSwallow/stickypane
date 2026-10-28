package form

import (
	"encoding/json"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/LeeSwallow/stickypane/internal/doc"
	"github.com/LeeSwallow/stickypane/internal/widget"
)

func pageOf(t *testing.T, d doc.Document) Page {
	t.Helper()
	b, err := json.Marshal(kind.Data(d, widget.DefaultEnv()))
	if err != nil {
		t.Fatal(err)
	}
	var p Page
	if err := json.Unmarshal(b, &p); err != nil {
		t.Fatal(err)
	}
	return p
}

// lineWhere finds the first line of a kind with this text.
func lineWhere(t *testing.T, p Page, kind, text string) int {
	t.Helper()
	for i, l := range p.Lines {
		if l.Kind == kind && l.Text == text {
			return i
		}
	}
	t.Fatalf("no %s line %q in %+v", kind, text, p.Lines)
	return -1
}

func TestDataOfAForm(t *testing.T) {
	p := pageOf(t, doc.Parse([]byte(deploy)))
	if p.Submitted != "" || p.At != "" {
		t.Errorf("submitted = %q at %q, want neither", p.Submitted, p.At)
	}
	i := lineWhere(t, p, "option", "production")
	want := Line{Kind: "option", Text: "production", Radio: true, Group: "어디에 배포할까요?"}
	if !reflect.DeepEqual(p.Lines[i], want) {
		t.Errorf("line = %+v, want %+v", p.Lines[i], want)
	}
	if l := p.Lines[lineWhere(t, p, "option", "캐시 비우기")]; !l.Checked || l.Radio {
		t.Errorf("checkbox = %+v", l)
	}
	if l := p.Lines[lineWhere(t, p, "detail", "먼저 올려 보고 확인합니다.")]; l.Kind != "detail" {
		t.Errorf("detail = %+v", l)
	}
	var labels []string
	for _, l := range p.Lines {
		if l.Kind == "buttons" {
			labels = l.Labels
		}
	}
	if !reflect.DeepEqual(labels, []string{"제출", "취소"}) {
		t.Errorf("labels = %v", labels)
	}
	b, _ := json.Marshal(kind.Data(doc.Parse([]byte(deploy)), widget.DefaultEnv()))
	var raw map[string]any
	_ = json.Unmarshal(b, &raw)
	for _, k := range []string{"lines", "submitted", "at"} {
		if _, ok := raw[k]; !ok {
			t.Errorf("Data has no %q: %s", k, b)
		}
	}
}

// sentAt is the clock the actions of these tests are taken by.
func sentAt() time.Time { return time.Date(2026, 10, 28, 14, 2, 11, 0, time.FixedZone("KST", 9*3600)) }

func doAct(t *testing.T, d doc.Document, name string, args map[string]string) (doc.Document, error) {
	t.Helper()
	op, err := kind.Act(d, widget.Action{Name: name, Args: args}, widget.Env{Clock: sentAt})
	if err != nil {
		if op != nil {
			t.Errorf("Op = %v with an error", op)
		}
		return d, err
	}
	out, err := op.Apply(d)
	if err != nil {
		t.Fatal(err)
	}
	return out, nil
}

func idx(i int) map[string]string { return map[string]string{"line": itoa(i)} }

func itoa(i int) string { b, _ := json.Marshal(i); return string(b) }

func TestActChooseThenSubmit(t *testing.T) {
	d := doc.Parse([]byte(deploy))

	p := pageOf(t, d)
	d, err := doAct(t, d, "choose", idx(lineWhere(t, p, "option", "production")))
	if err != nil {
		t.Fatal(err)
	}
	f := idx(lineWhere(t, p, "field", ""))
	f["text"] = "after lunch"
	if d, err = doAct(t, d, "fill", f); err != nil {
		t.Fatal(err)
	}
	if d, err = doAct(t, d, "submit", map[string]string{"label": "제출"}); err != nil {
		t.Fatal(err)
	}
	got := Read(d)
	want := Answers{Submitted: true, Button: "제출", At: "2026-10-28T14:02:11+09:00", Answers: []Answer{
		{Question: "어디에 배포할까요?", Values: []string{"production"}},
		{Question: "같이 할 일", Values: []string{"캐시 비우기"}},
		{Question: "메모", Values: []string{"after lunch"}},
	}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Answers = %+v\nwant     %+v", got, want)
	}
	if p := pageOf(t, d); p.Submitted != "제출" || p.At != want.At {
		t.Errorf("Data submitted %q at %q", p.Submitted, p.At)
	}
}

func TestActChooseOutOfRange(t *testing.T) {
	d := doc.Parse([]byte(deploy))
	p := pageOf(t, d)
	for _, bad := range []map[string]string{
		{"line": "-1"}, {"line": "999"}, {"line": "x"}, {},
		idx(lineWhere(t, p, "text", "배포 전에 확인합니다.")),
		idx(lineWhere(t, p, "buttons", "")),
	} {
		if _, err := doAct(t, d, "choose", bad); err == nil {
			t.Errorf("choose %v succeeded", bad)
		}
	}
	for _, bad := range []map[string]string{{"line": "0", "text": "x"}, {"line": "999", "text": "x"}, {"line": itoa(lineWhere(t, p, "field", "")), "text": "a\nb"}, {"line": itoa(lineWhere(t, p, "field", ""))}} {
		if _, err := doAct(t, d, "fill", bad); err == nil {
			t.Errorf("fill %v succeeded", bad)
		}
	}
	if _, err := doAct(t, d, "submit", map[string]string{"label": "없는 버튼"}); err == nil {
		t.Error("submit of a missing button succeeded")
	}
}

func TestActOnASubmittedFormStillWorks(t *testing.T) {
	d, err := doAct(t, doc.Parse([]byte(deploy)), "submit", map[string]string{"label": "제출"})
	if err != nil {
		t.Fatal(err)
	}
	p := pageOf(t, d)
	d, err = doAct(t, d, "choose", idx(lineWhere(t, p, "option", "staging")))
	if err != nil {
		t.Fatal(err)
	}
	if a := Read(d); a.Submitted || a.At != "" {
		t.Errorf("a changed answer keeps the submission: %+v", a)
	}
}

func TestActFillSameTextKeepsSubmission(t *testing.T) {
	d, _ := doAct(t, doc.Parse([]byte(deploy)), "submit", map[string]string{"label": "제출"})
	p := pageOf(t, d)
	d, err := doAct(t, d, "fill", map[string]string{"line": itoa(lineWhere(t, p, "field", "")), "text": ""})
	if err != nil {
		t.Fatal(err)
	}
	if !Read(d).Submitted {
		t.Error("filling the same text took the submission back")
	}
}

func TestActUnknownAction(t *testing.T) {
	_, err := kind.Act(doc.Parse([]byte(deploy)), widget.Action{Name: "dance"}, widget.DefaultEnv())
	if !errors.Is(err, widget.ErrUnknownAction) {
		t.Errorf("err = %v", err)
	}
}

func TestActSubmitTheBuiltInButton(t *testing.T) {
	d, err := doAct(t, doc.Parse([]byte("---\ntype: form\n---\n## Q\n- ( ) a\n")), "submit", map[string]string{"label": defaultButton})
	if err != nil || !Read(d).Submitted {
		t.Errorf("err = %v, answers = %+v", err, Read(d))
	}
}
