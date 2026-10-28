package board

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/LeeSwallow/stickypane/internal/doc"
	"github.com/LeeSwallow/stickypane/internal/when"
	"github.com/LeeSwallow/stickypane/internal/widget"
)

type cardData struct {
	Text    string   `json:"text"`
	Who     []string `json:"who"`
	Moved   string   `json:"moved"`
	Detail  []string `json:"detail"`
	Stalled bool     `json:"stalled"`
}

type boardData struct {
	Desc    []string `json:"desc"`
	Columns []struct {
		Title string     `json:"title"`
		Cards []cardData `json:"cards"`
	} `json:"columns"`
}

// dataOf is the board's Data at 15:00 on the day its cards moved.
func dataOf(t *testing.T, d doc.Document) boardData {
	t.Helper()
	return dataAt(t, d, 15, 0)
}

func dataAt(t *testing.T, d doc.Document, h, m int) boardData {
	t.Helper()
	b, err := json.Marshal(Kind.Data(d, envAt(h, m)))
	if err != nil {
		t.Fatal(err)
	}
	var out boardData
	if err := json.Unmarshal(b, &out); err != nil {
		t.Fatal(err)
	}
	return out
}

func doAct(t *testing.T, src, name string, args map[string]string) (doc.Document, error) {
	t.Helper()
	d := doc.Parse([]byte(src))
	op, err := Kind.Act(d, widget.Action{Name: name, Args: args}, widget.DefaultEnv())
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

func titles(b boardData, col int) string {
	var s []string
	for _, c := range b.Columns[col].Cards {
		s = append(s, c.Text)
	}
	return strings.Join(s, ",")
}

func TestActMoveCardToAnotherColumn(t *testing.T) {
	t.Parallel()
	d, err := doAct(t, Kind.Example, "move", map[string]string{"col": "1", "i": "0", "to": "2", "at": "0"})
	if err != nil {
		t.Fatal(err)
	}
	b := dataOf(t, d)
	if got := titles(b, 1); got != "" {
		t.Errorf("Doing = %q, want empty", got)
	}
	if got := titles(b, 2); got != "login API,schema" {
		t.Errorf("Done = %q, want login API first", got)
	}
	// Without at the card goes to the end.
	d, err = doAct(t, Kind.Example, "move", map[string]string{"col": "1", "i": "0", "to": "2"})
	if err != nil {
		t.Fatal(err)
	}
	if got := titles(dataOf(t, d), 2); got != "schema,login API" {
		t.Errorf("Done = %q, want login API last", got)
	}
}

func TestActMoveOutOfRange(t *testing.T) {
	t.Parallel()
	for _, args := range []map[string]string{
		{"col": "9", "i": "0", "to": "2"},
		{"col": "1", "i": "5", "to": "2"},
		{"col": "1", "i": "-1", "to": "2"},
		{"col": "1", "i": "0", "to": "3"},
		{"col": "1", "i": "0", "to": "2", "at": "2"},
		{"col": "1", "i": "x", "to": "2"},
		{"col": "1", "i": "0", "to": "2", "at": "x"},
	} {
		if _, err := doAct(t, Kind.Example, "move", args); err == nil {
			t.Errorf("move %v: no error", args)
		}
	}
}

func TestActAddAndUnknown(t *testing.T) {
	t.Parallel()
	d, err := doAct(t, Kind.Example, "add", map[string]string{"col": "0", "text": "docs"})
	if err != nil {
		t.Fatal(err)
	}
	if got := titles(dataOf(t, d), 0); got != "payments,docs" {
		t.Errorf("To do = %q", got)
	}
	for _, args := range []map[string]string{{"col": "7", "text": "x"}, {"col": "0", "text": " "}, {"col": "0", "text": "a\nb"}} {
		if _, err := doAct(t, Kind.Example, "add", args); err == nil {
			t.Errorf("add %v: no error", args)
		}
	}
	if _, err := doAct(t, Kind.Example, "frob", nil); !errors.Is(err, widget.ErrUnknownAction) {
		t.Errorf("err = %v, want ErrUnknownAction", err)
	}
}

func TestActMoveSameTextCards(t *testing.T) {
	t.Parallel()
	src := "## A\n- x\n- x\n## B\n- x\n"
	d, err := doAct(t, src, "move", map[string]string{"col": "0", "i": "1", "to": "1", "at": "0"})
	if err != nil {
		t.Fatal(err)
	}
	b := dataOf(t, d)
	if titles(b, 0) != "x" || titles(b, 1) != "x,x" {
		t.Errorf("got %q / %q", titles(b, 0), titles(b, 1))
	}
}

func TestDataMarksAStalledCard(t *testing.T) {
	t.Parallel()
	src := "Plan\n## Doing\n- old @alice @{2026-10-03} @@{14:00}\n  note\n- fresh @{2026-10-03} @@{14:50}\n## Done\n- gone @{2026-10-03} @@{09:00}\n"
	b := dataOf(t, doc.Parse([]byte(src)))
	if len(b.Desc) != 1 || b.Desc[0] != "Plan" {
		t.Errorf("desc = %v", b.Desc)
	}
	c := b.Columns[0].Cards
	if !c[0].Stalled || c[1].Stalled || b.Columns[1].Cards[0].Stalled {
		t.Errorf("stalled = %v %v %v", c[0].Stalled, c[1].Stalled, b.Columns[1].Cards[0].Stalled)
	}
	if c[0].Who[0] != "alice" || c[0].Detail[0] != "note" || c[0].Moved == "" {
		t.Errorf("card = %+v", c[0])
	}
	if raw, _ := json.Marshal(Kind.Data(doc.Parse(Kind.Template("B")), widget.DefaultEnv())); !strings.Contains(string(raw), `"desc":[]`) {
		t.Errorf("empty board = %s", raw)
	}
}

func TestDataWritesMovedInTheMachineLayout(t *testing.T) {
	t.Parallel()
	b := dataOf(t, doc.Parse([]byte("## Doing\n- a @{2026-10-03} @@{14:02}\n")))
	if got := b.Columns[0].Cards[0].Moved; got != at(14, 2).Format(when.Moment) {
		t.Errorf("moved = %q", got)
	}
}

func TestDataStaleBoundary(t *testing.T) {
	t.Parallel()
	src := "## Doing\n- a @{2026-10-03} @@{14:30}\n## Done\n"
	for _, c := range []struct {
		h, m int
		want bool
	}{{14, 59, false}, {15, 0, true}} {
		if got := dataAt(t, doc.Parse([]byte(src)), c.h, c.m).Columns[0].Cards[0].Stalled; got != c.want {
			t.Errorf("at %d:%02d stalled = %v, want %v", c.h, c.m, got, c.want)
		}
	}
}

func TestActMoveIntoTheMiddleAmongSameText(t *testing.T) {
	t.Parallel()
	src := "## A\n- x\n## B\n- p\n- x\n- q\n"
	d, err := doAct(t, src, "move", map[string]string{"col": "0", "i": "0", "to": "1", "at": "2"})
	if err != nil {
		t.Fatal(err)
	}
	if got := titles(dataOf(t, d), 1); got != "p,x,x,q" {
		t.Errorf("B = %q", got)
	}
	d, err = doAct(t, src, "move", map[string]string{"col": "0", "i": "0", "to": "1", "at": "1"})
	if err != nil {
		t.Fatal(err)
	}
	if got := titles(dataOf(t, d), 1); got != "p,x,x,q" {
		t.Errorf("B = %q", got)
	}
}

func TestActMoveWithinAColumn(t *testing.T) {
	t.Parallel()
	src := "## A\n- a\n- b\n- c\n"
	for at, want := range map[string]string{"0": "c,a,b", "1": "a,c,b", "2": "a,b,c"} {
		d, err := doAct(t, src, "move", map[string]string{"col": "0", "i": "2", "to": "0", "at": at})
		if err != nil {
			t.Fatal(err)
		}
		if got := titles(dataOf(t, d), 0); got != want {
			t.Errorf("at %s: %q, want %q", at, got, want)
		}
	}
	if _, err := doAct(t, src, "move", map[string]string{"col": "0", "i": "2", "to": "0", "at": "3"}); err == nil {
		t.Error("at past the column: no error")
	}
}
