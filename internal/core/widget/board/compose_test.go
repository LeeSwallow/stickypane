package board

import (
	"errors"
	"strings"
	"testing"

	"github.com/LeeSwallow/stickypane/internal/core/doc"
	"github.com/LeeSwallow/stickypane/internal/core/widget"
)

func TestComposeWritesColumnsAndFirstCards(t *testing.T) {
	t.Parallel()
	b, err := Kind.Make("Sprint", map[string]string{
		"columns": "Backlog\nDoing\nShipped",
		"cards":   "payments @min\nDoing: login API @codex-1\nshipped: schema",
	}, widget.Env{})
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
	t.Parallel()
	b, err := Kind.Make("Work", nil, widget.Env{})
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
	t.Parallel()
	_, err := Kind.Make("x", map[string]string{"columns": " \n "}, widget.Env{})
	if !errors.Is(err, widget.ErrBadOption) || !strings.Contains(err.Error(), "column") {
		t.Errorf("err = %v", err)
	}
}

// korean is an Env in a language whose words for the usual columns are
// not English.
func korean() widget.Env {
	words := map[string]string{"To do": "할 일", "Doing": "하는 중", "Done": "끝"}
	return widget.Env{Translate: func(p string) string {
		if w, ok := words[p]; ok {
			return w
		}
		return p
	}}
}

func titlesOf(b []byte) string {
	var titles []string
	for _, c := range read(doc.Parse(b)).cols {
		titles = append(titles, c.title)
	}
	return strings.Join(titles, "|")
}

// A new board's columns are in the words of the Env it is made with, from
// its template and from its create options alike; the form shows them so.
func TestANewBoardsColumnsAreInTheEnvsWords(t *testing.T) {
	t.Parallel()
	if got := titlesOf(Kind.Template("Work", korean())); got != "할 일|하는 중|끝" {
		t.Errorf("Template: columns = %v", got)
	}
	b, err := Kind.Make("Work", map[string]string{"cards": "x"}, korean())
	if err != nil {
		t.Fatal(err)
	}
	if got := titlesOf(b); got != "할 일|하는 중|끝" || read(doc.Parse(b)).cols[0].cards[0].text != "x" {
		t.Errorf("Make: %q", b)
	}
	if got := Kind.OptionsIn(korean())[0].Default; got != "할 일\n하는 중\n끝" {
		t.Errorf("the form's default columns = %q", got)
	}
	if got := Kind.Options[0].Default; got != "To do\nDoing\nDone" {
		t.Errorf("the English default = %q", got)
	}
}

// A board started for a first card has the usual columns when the card
// goes to one of them, or to none named, in English when the column is
// named in English; else that card's column alone.
func TestStartingABoardForACard(t *testing.T) {
	t.Parallel()
	for _, c := range []struct {
		to   string
		env  widget.Env
		want string
	}{
		{"", korean(), "할 일|하는 중|끝"},
		{"하는 중", korean(), "할 일|하는 중|끝"},
		{"끝", korean(), "할 일|하는 중|끝"},
		{"진행 중", korean(), "진행 중"},
		{"Doing", korean(), "To do|Doing|Done"},
		{"done", korean(), "To do|Doing|Done"},
		{"doing", widget.Env{}, "To do|Doing|Done"},
		{" Backlog ", widget.Env{}, "Backlog"},
	} {
		if got := titlesOf(Starting("Work", c.to, c.env)); got != c.want {
			t.Errorf("Starting(%q) = %v, want %v", c.to, got, c.want)
		}
	}
}

// A card added without a column to a board that has none makes the first
// column, in the board's words when the caller says them.
func TestAddingToABoardWithoutColumns(t *testing.T) {
	t.Parallel()
	d := doc.Parse(widget.NewFile("board", "B", ""))
	for _, c := range []struct{ first, want string }{{"", "## To do\n- x\n"}, {"할 일", "## 할 일\n- x\n"}} {
		got, err := Add{Card: "x", First: c.first}.Apply(d)
		if err != nil || got.Body != c.want {
			t.Errorf("First %q: body = %q, %v", c.first, got.Body, err)
		}
	}
}
