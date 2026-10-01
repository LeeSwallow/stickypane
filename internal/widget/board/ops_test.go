package board

import (
	"errors"
	"testing"

	"github.com/LeeSwallow/stickypane/internal/doc"
)

const duplicates = "## A\n- same\n  first detail\n- same\n  second detail\n- z\n## B\n"

func apply(t *testing.T, op doc.Op, body string) string {
	t.Helper()
	d, err := op.Apply(doc.Document{Body: body})
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}
	return d.Body
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

func TestOpsConflict(t *testing.T) {
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

func TestMoveCardPicksTheRightDuplicate(t *testing.T) {
	got := apply(t, MoveCard{From: "A", To: "B", Text: "same", Nth: 1}, duplicates)
	want := "## A\n- same\n  first detail\n- z\n## B\n- same\n  second detail\n"
	if got != want {
		t.Errorf("got %q\nwant %q", got, want)
	}
}

func TestReorderCardPicksTheRightDuplicate(t *testing.T) {
	got := apply(t, ReorderCard{Col: "A", Text: "same", Nth: 1, Delta: 1}, duplicates)
	want := "## A\n- same\n  first detail\n- z\n- same\n  second detail\n## B\n"
	if got != want {
		t.Errorf("got %q\nwant %q", got, want)
	}
}

func TestMissingDuplicateIsConflict(t *testing.T) {
	for _, op := range []doc.Op{
		MoveCard{From: "A", To: "B", Text: "same", Nth: 2},
		ReorderCard{Col: "A", Text: "same", Nth: 2, Delta: -1},
	} {
		d, err := op.Apply(doc.Document{Body: duplicates})
		if !errors.Is(err, doc.ErrConflict) || d.Body != duplicates {
			t.Errorf("%+v: err = %v, body changed = %v", op, err, d.Body != duplicates)
		}
	}
}

func TestMoveCardKeepsDetailsAcrossBlankLines(t *testing.T) {
	src := "## A\n- x\n  ```\n  code\n\n  more\n  ```\n- y\n## B\n"
	got := apply(t, MoveCard{From: "A", To: "B", Text: "x"}, src)
	want := "## A\n- y\n## B\n- x\n  ```\n  code\n\n  more\n  ```\n"
	if got != want {
		t.Errorf("got %q\nwant %q", got, want)
	}
}

func TestBlankLineBetweenCardsStaysPut(t *testing.T) {
	got := apply(t, MoveCard{From: "A", To: "B", Text: "x"}, "## A\n- x\n\n- y\n## B\n")
	if want := "## A\n\n- y\n## B\n- x\n"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestMoveCardThatIsNotAListItem(t *testing.T) {
	src := "## To do\nask Bob about the API\n1. first step\n- normal card\n## Done\n"
	got := apply(t, MoveCard{From: "To do", To: "Done", Text: "ask Bob about the API"}, src)
	if want := "## To do\n1. first step\n- normal card\n## Done\nask Bob about the API\n"; got != want {
		t.Errorf("got %q\nwant %q", got, want)
	}
}
