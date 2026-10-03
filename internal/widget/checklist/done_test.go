package checklist

import (
	"strings"
	"testing"
	"time"

	"github.com/LeeSwallow/stickypane/internal/widget"

	"github.com/charmbracelet/x/ansi"

	"github.com/LeeSwallow/stickypane/internal/doc"
)

const at = "2026-10-03 14:02"

func applyOp(t *testing.T, src string, op doc.Op) string {
	t.Helper()
	d, err := op.Apply(doc.Parse([]byte(src)))
	if err != nil {
		t.Fatalf("%T: %v", op, err)
	}
	return string(d.Bytes())
}

// Ticking an item writes when it was done; unticking takes it back out.
func TestTickingAnItemWritesWhenItWasDone(t *testing.T) {
	src := "- [ ] a\n- [ ] b\r\n"
	got := applyOp(t, src, Toggle{Text: "b", Checked: true, At: at})
	if got != "- [ ] a\n- [x] b ✅ 2026-10-03 14:02\r\n" {
		t.Fatalf("ticked = %q", got)
	}
	if back := applyOp(t, got, Toggle{Text: "b", Checked: false}); back != src {
		t.Errorf("unticked = %q, want %q", back, src)
	}
	if got := applyOp(t, src, Check{Item: "#2", Checked: true, At: at}); !strings.Contains(got, "- [x] b ✅ 2026-10-03 14:02") {
		t.Errorf("Check by position = %q", got)
	}
	if got := applyOp(t, "- [x] b ✅ 2026-10-01 09:00\n", Check{Item: "b", Checked: true, At: at}); got != "- [x] b ✅ 2026-10-01 09:00\n" {
		t.Errorf("an item already done keeps its time: %q", got)
	}
	if got := applyOp(t, "- [ ] a\n", Toggle{Text: "a", Checked: true}); got != "- [x] a\n" {
		t.Errorf("without a time nothing is added: %q", got)
	}
}

// The time is not part of the item's name: an item is found by its text.
func TestTheTimeIsNotPartOfTheName(t *testing.T) {
	c := parse(doc.Parse([]byte("- [x] write tests ✅ 2026-10-02\n")))
	if e := c.entries[c.items[0]]; e.text != "write tests" || e.done.Format("2006-01-02") != "2026-10-02" {
		t.Errorf("entry = %+v", e)
	}
	if got := applyOp(t, "- [x] write tests ✅ 2026-10-02\n", Check{Item: "write tests", Checked: false}); got != "- [ ] write tests\n" {
		t.Errorf("unchecked by name = %q", got)
	}
}

// The screen shows what is left first, then what is done, the most recent
// first, with when.
func TestOpenAndDoneItemsAreShownApart(t *testing.T) {
	now = func() time.Time { return time.Date(2026, 10, 3, 18, 0, 0, 0, time.Local) }
	t.Cleanup(func() { now = func() time.Time { return widget.Now() } })
	src := "- [x] old ✅ 2026-09-30 10:00\n- [ ] first\n- [x] recent ✅ 2026-10-03 14:02\n- [ ] second\n- [x] undated\n"
	c := parse(doc.Parse([]byte(src)))
	out, _ := c.Draw(50, false)
	lines := strings.Split(ansi.Strip(out), "\n")
	var order []string
	for _, l := range lines {
		for _, name := range []string{"first", "second", "Done (3)", "recent", "old", "undated"} {
			if strings.Contains(l, name) {
				order = append(order, name)
			}
		}
	}
	if strings.Join(order, ",") != "first,second,Done (3),recent,old,undated" {
		t.Fatalf("order = %v\n%s", order, ansi.Strip(out))
	}
	for _, l := range lines {
		if strings.Contains(l, "recent") && !strings.HasSuffix(strings.TrimRight(l, " "), "14:02") {
			t.Errorf("today's item shows its time: %q", l)
		}
		if strings.Contains(l, "old") && !strings.HasSuffix(strings.TrimRight(l, " "), "Sep 30") {
			t.Errorf("an older item shows its day: %q", l)
		}
	}
}

// The cursor walks the screen's order, and ticking stamps the time.
func TestTheCursorFollowsTheScreenAndTickingStampsNow(t *testing.T) {
	now = func() time.Time { return time.Date(2026, 10, 3, 14, 2, 0, 0, time.Local) }
	t.Cleanup(func() { now = func() time.Time { return widget.Now() } })
	c := parse(doc.Parse([]byte("- [x] done ✅ 2026-10-01 09:00\n- [ ] todo\n")))
	_, res := c.Update("space") // the first on the screen is the open one
	op, ok := res.Op.(Toggle)
	if !ok || op.Text != "todo" || !op.Checked || op.At != at {
		t.Fatalf("op = %#v", res.Op)
	}
	c = parse(doc.Parse([]byte("- [x] done ✅ 2026-10-01 09:00\n- [ ] todo\n")))
	c.Update("j")
	if _, res := c.Update("space"); res.Op.(Toggle).Text != "done" || res.Op.(Toggle).Checked {
		t.Errorf("j moves into the done items: %#v", res.Op)
	}
}

// An item ticked by editing the file, without a time, gets one from
// stickypane: the time the file changed. Items that have one keep it.
func TestTendStampsItemsTickedWithoutATime(t *testing.T) {
	changed := time.Date(2026, 10, 3, 9, 30, 0, 0, time.Local)
	d := doc.Parse([]byte("- [x] a\n- [ ] b\n- [x] c ✅ 2026-10-01 08:00\r\n"))
	op := Kind.Tend(doc.Document{}, d, changed)
	if op == nil {
		t.Fatal("an item ticked without a time needs one")
	}
	out, err := op.Apply(d)
	if err != nil {
		t.Fatal(err)
	}
	if got := string(out.Bytes()); got != "- [x] a ✅ 2026-10-03 09:30\n- [ ] b\n- [x] c ✅ 2026-10-01 08:00\r\n" {
		t.Errorf("tended = %q", got)
	}
	if Kind.Tend(doc.Document{}, out, changed) != nil {
		t.Error("a list with every time written needs nothing")
	}
}
