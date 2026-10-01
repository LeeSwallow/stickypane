package board

import (
	"errors"
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"

	"github.com/LeeSwallow/stickypane/internal/doc"
)

const duplicates = "## A\n- same\n  first detail\n- same\n  second detail\n- z\n## B\n"

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

func TestKeysCarryTheDuplicateIndex(t *testing.T) {
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

func TestMoveCardKeepsDetailsAcrossBlankLines(t *testing.T) {
	body := "## A\n- x\n  ```\n  code\n\n  more\n  ```\n- y\n## B\n"
	got := apply(t, MoveCard{From: "A", To: "B", Text: "x"}, body)
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

func TestLinesThatAreNotListItemsAreStillCards(t *testing.T) {
	src := "## To do\nask Bob about the API\n1. first step\n- normal card\n## Done\n"
	view := ansi.Strip(parseSrc(src).View(70, 12))
	for _, want := range []string{"To do (3)", "ask Bob about the API", "1. first step", "normal card"} {
		if !strings.Contains(view, want) {
			t.Errorf("View should contain %q:\n%s", want, view)
		}
	}
	got := apply(t, MoveCard{From: "To do", To: "Done", Text: "ask Bob about the API"}, src)
	if want := "## To do\n1. first step\n- normal card\n## Done\nask Bob about the API\n"; got != want {
		t.Errorf("got %q\nwant %q", got, want)
	}
}

func TestViewShowsTheDescription(t *testing.T) {
	w := parseSrc("Sprint goal: ship login\n\n## A\n- x\n")
	if view := ansi.Strip(w.View(60, 12)); !strings.Contains(view, "Sprint goal: ship login") {
		t.Errorf("the open board should show its description:\n%s", view)
	}
	if preview := ansi.Strip(w.Preview(60)); strings.Contains(preview, "Sprint goal") {
		t.Errorf("the description belongs to the open board only:\n%s", preview)
	}
}

func TestBoardWithoutColumnsStillShowsItsText(t *testing.T) {
	w := parseSrc("just text\nmore text\n")
	for name, out := range map[string]string{"Preview": w.Preview(60), "View": w.View(60, 10)} {
		plain := ansi.Strip(out)
		for _, want := range []string{`"## Name"`, "just text", "more text"} {
			if !strings.Contains(plain, want) {
				t.Errorf("%s should contain %q:\n%s", name, want, plain)
			}
		}
	}
}
