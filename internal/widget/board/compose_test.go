package board

import (
	"errors"
	"strings"
	"testing"

	"github.com/LeeSwallow/stickypane/internal/doc"
	"github.com/LeeSwallow/stickypane/internal/widget"
)

func TestComposeWritesColumnsAndFirstCards(t *testing.T) {
	b, err := Kind.Make("Sprint", map[string]string{
		"columns": "Backlog\nDoing\nShipped",
		"cards":   "payments @min\nDoing: login API @codex-1\nshipped: schema",
	})
	if err != nil {
		t.Fatal(err)
	}
	d := doc.Parse(b)
	if d.Type() != "board" {
		t.Errorf("type = %q", d.Type())
	}
	if title, _ := d.Get("title"); title != "Sprint" {
		t.Errorf("title = %q", title)
	}
	got := read(d)
	want := [][]string{{"Backlog", "payments"}, {"Doing", "login API"}, {"Shipped", "schema"}}
	if len(got.cols) != len(want) {
		t.Fatalf("columns = %+v", got.cols)
	}
	for i, w := range want {
		c := got.cols[i]
		if c.title != w[0] || len(c.cards) != 1 || c.cards[0].text != w[1] {
			t.Errorf("column %d = %+v, want %v", i, c, w)
		}
	}
	if who := got.cols[1].cards[0].who; len(who) != 1 || who[0] != "codex-1" {
		t.Errorf("who has the login card: %v", who)
	}
}

func TestComposeDefaultsToTheTemplateColumns(t *testing.T) {
	b, err := Kind.Make("Work", nil)
	if err != nil {
		t.Fatal(err)
	}
	var titles []string
	for _, c := range read(doc.Parse(b)).cols {
		titles = append(titles, c.title)
	}
	if strings.Join(titles, "|") != "To do|Doing|Done" {
		t.Errorf("columns = %v", titles)
	}
}

func TestComposeRefusesABoardWithoutColumns(t *testing.T) {
	_, err := Kind.Make("x", map[string]string{"columns": " \n "})
	if !errors.Is(err, widget.ErrBadOption) || !strings.Contains(err.Error(), "column") {
		t.Errorf("err = %v", err)
	}
}
