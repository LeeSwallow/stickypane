package kinds

import (
	"fmt"
	"strings"
	"testing"

	"github.com/LeeSwallow/stickypane/internal/doc"
	"github.com/LeeSwallow/stickypane/internal/widget"
	"github.com/LeeSwallow/stickypane/internal/widget/note"
)

func events(t *testing.T, kind, before, after string) string {
	t.Helper()
	k := Default(note.Plain).Lookup(kind)
	if k.Events == nil {
		t.Fatalf("%s has no events", kind)
	}
	var out []string
	for _, e := range k.Events(doc.Parse([]byte(before)), doc.Parse([]byte(after))) {
		out = append(out, fmt.Sprintf("%s %q %q→%q", e.Type, e.Item, e.From, e.To))
	}
	return strings.Join(out, "; ")
}

// Each shape says what happened in it, in its own words, so that anything
// watching the board can react to it.
func TestEveryShapeSaysWhatHappened(t *testing.T) {
	for _, c := range []struct{ kind, before, after, want string }{
		{"checklist", "- [ ] a\n- [ ] b\n", "- [x] a ✅ 2026-10-03 14:02\n- [ ] b\n- [ ] c\n",
			`item.ticked "a" ""→""; item.added "c" ""→""`},
		{"checklist", "- [x] a\n", "- [ ] a\n", `item.unticked "a" ""→""`},
		{"board", "## To do\n- a\n- b\n## Done\n", "## To do\n- b @x\n## Done\n- a @{2026-10-03} @@{14:02}\n- c\n",
			`card.moved "a" "To do"→"Done"; card.added "c" ""→"Done"`},
		{"board", "## To do\n- a\n", "## To do\n", `card.removed "a" "To do"→""`},
		{"form", "---\ntype: form\n---\n- ( ) x\n", "---\ntype: form\nsubmitted: OK\n---\n- (x) x\n",
			`form.submitted "OK" ""→""`},
		{"log", "one\n", "one\ntwo\nthree\n", `log.appended "two" ""→""; log.appended "three" ""→""`},
		{"chart", "a: 1\nb: 2\n", "a: 3\nb: 2\nc: 5\n", `chart.changed "a" "1"→"3"; chart.changed "c" ""→"5"`},
	} {
		if got := events(t, c.kind, c.before, c.after); got != c.want {
			t.Errorf("%s:\n got %s\nwant %s", c.kind, got, c.want)
		}
	}
	var _ widget.Event // the type the shapes share
}
