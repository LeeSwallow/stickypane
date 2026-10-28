package chart

import (
	"errors"
	"strings"
	"testing"

	"github.com/LeeSwallow/stickypane/internal/doc"
	"github.com/LeeSwallow/stickypane/internal/widget"
)

func TestComposeWritesTheViewAndValues(t *testing.T) {
	b, err := Kind.Make("Tokens", map[string]string{"view": "spark", "values": "mon: 3\ntue: 1,200\n\nwed: -2.5"})
	if err != nil {
		t.Fatal(err)
	}
	d := doc.Parse(b)
	got := data(d).(Data)
	if d.Type() != "chart" || got.View != "spark" {
		t.Errorf("type %q, view %q", d.Type(), got.View)
	}
	var labels []string
	for _, p := range got.Points {
		labels = append(labels, p.Label+"="+p.Text)
	}
	if strings.Join(labels, " ") != "mon=3 tue=1,200 wed=-2.5" {
		t.Errorf("points = %v", labels)
	}
}

func TestComposeCountsFromAnotherNote(t *testing.T) {
	b, err := Kind.Make("Progress", map[string]string{"from": "plan"})
	if err != nil {
		t.Fatal(err)
	}
	d := doc.Parse(b)
	if from, _ := d.Get("from"); from != "plan" {
		t.Errorf("from = %q in %q", from, b)
	}
	if _, ok := d.Get("view"); ok {
		t.Errorf("bars are the default and need no view: %q", b)
	}
}

func TestComposeRefusesALineThatIsNotAValue(t *testing.T) {
	_, err := Kind.Make("x", map[string]string{"values": "mon: 3\nlots"})
	if !errors.Is(err, widget.ErrBadOption) || !strings.Contains(err.Error(), "lots") {
		t.Errorf("err = %v", err)
	}
	if _, err := Kind.Make("x", map[string]string{"view": "pie"}); !errors.Is(err, widget.ErrBadOption) {
		t.Errorf("view pie: %v", err)
	}
}
