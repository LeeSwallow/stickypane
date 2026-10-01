package checklist

import (
	"errors"
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"

	"github.com/LeeSwallow/stickypane/internal/doc"
)

const sameItems = "## Backend\n- [ ] tests\n## Frontend\n- [ ] tests\n"

func TestTogglePicksTheRightDuplicate(t *testing.T) {
	got := apply(t, Toggle{Text: "tests", Checked: true, Nth: 1}, sameItems)
	if want := "## Backend\n- [ ] tests\n## Frontend\n- [x] tests\n"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestToggleOfMissingDuplicateIsConflict(t *testing.T) {
	d, err := Toggle{Text: "tests", Checked: true, Nth: 2}.Apply(doc.Document{Body: sameItems})
	if !errors.Is(err, doc.ErrConflict) || d.Body != sameItems {
		t.Errorf("err = %v, body changed = %v", err, d.Body != sameItems)
	}
}

func TestSpaceCarriesTheDuplicateIndex(t *testing.T) {
	w := parseBody(sameItems)
	w, _ = w.Update("j")
	_, res := w.Update("space")
	if want := (Toggle{Text: "tests", Checked: true, Nth: 1}); res.Op != want {
		t.Errorf("Op = %+v, want %+v", res.Op, want)
	}
}

func TestAddItemGoesAfterTheLastItemsDetails(t *testing.T) {
	got := apply(t, AddItem{Text: "new"}, "- [ ] a\n  detail of a\n\nnotes\n")
	if want := "- [ ] a\n  detail of a\n- [ ] new\n\nnotes\n"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestChecklistWithoutItemsStillShowsItsText(t *testing.T) {
	w := parseBody("# Steps\n1. [ ] one\n[ ] another\n")
	for name, out := range map[string]string{"Preview": w.Preview(50), "View": w.View(50, 10)} {
		plain := ansi.Strip(out)
		for _, want := range []string{`"- [ ] task"`, "# Steps", "1. [ ] one", "[ ] another"} {
			if !strings.Contains(plain, want) {
				t.Errorf("%s should contain %q:\n%s", name, want, plain)
			}
		}
	}
}

func TestNewChecklistStartsEmpty(t *testing.T) {
	w := Kind.Parse(doc.Parse(Kind.Template("Release")))
	if got := ansi.Strip(w.Preview(50)); strings.Contains(got, "0/1") {
		t.Errorf("a new checklist must not contain a blank item: %q", got)
	}
	if got := apply(t, AddItem{Text: "ship"}, doc.Parse(Kind.Template("Release")).Body); got != "- [ ] ship\n" {
		t.Errorf("first item: got %q", got)
	}
}
