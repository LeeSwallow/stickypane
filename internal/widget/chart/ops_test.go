package chart

import (
	"testing"

	"github.com/LeeSwallow/stickypane/internal/doc"
)

func apply(t *testing.T, src string, op doc.Op) string {
	t.Helper()
	d, err := op.Apply(doc.Parse([]byte(src)))
	if err != nil {
		t.Fatalf("%+v: %v", op, err)
	}
	return string(d.Bytes())
}

func TestSetReplacesOrAddsAValue(t *testing.T) {
	src := "---\ntype: chart\n---\nTests per package\n\n- app: 61\nboard: 31\n"
	if got := apply(t, src, Set{Label: "app", Value: "67"}); got != "---\ntype: chart\n---\nTests per package\n\n- app: 67\nboard: 31\n" {
		t.Errorf("Set keeps the line as it was written: %q", got)
	}
	if got := apply(t, src, Set{Label: "store", Value: "17"}); got != src+"store: 17\n" {
		t.Errorf("a new label is added at the end: %q", got)
	}
	if got := apply(t, "a: 1", Set{Label: "b", Value: "2"}); got != "a: 1\nb: 2\n" {
		t.Errorf("a body without a final newline: %q", got)
	}
	if got := apply(t, "", Set{Label: "a", Value: "1"}); got != "a: 1\n" {
		t.Errorf("an empty chart: %q", got)
	}
	if _, err := (Set{Label: "a", Value: "many"}).Apply(doc.Parse([]byte(""))); err == nil {
		t.Error("a value that is not a number is an error")
	}
	if _, err := (Set{Label: "", Value: "1"}).Apply(doc.Parse([]byte(""))); err == nil {
		t.Error("a value needs a label")
	}
}

func TestAddCountsUp(t *testing.T) {
	if got := apply(t, "done: 3\nleft: 9\n", Add{Label: "done", Delta: 1}); got != "done: 4\nleft: 9\n" {
		t.Errorf("Add = %q", got)
	}
	if got := apply(t, "tokens: 1,200\n", Add{Label: "tokens", Delta: 800.5}); got != "tokens: 2,000.5\n" {
		t.Errorf("a value written with commas keeps them: %q", got)
	}
	if got := apply(t, "done: 3\n", Add{Label: "skipped", Delta: 2}); got != "done: 3\nskipped: 2\n" {
		t.Errorf("a label that is not there starts from zero: %q", got)
	}
	if got := apply(t, "left: 9\n", Add{Label: "left", Delta: -1}); got != "left: 8\n" {
		t.Errorf("counting down: %q", got)
	}
}
