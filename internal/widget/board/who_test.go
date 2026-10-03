package board

import (
	"strings"
	"testing"
	"time"

	"github.com/LeeSwallow/stickypane/internal/widget"

	"github.com/charmbracelet/x/ansi"

	"github.com/LeeSwallow/stickypane/internal/doc"
)

func at(h, m int) time.Time { return time.Date(2026, 10, 3, h, m, 0, 0, time.Local) }

// A card says who has it (Backlog.md's @name) and when it last moved
// (Obsidian Kanban's @{date} @@{time}); neither is part of its name.
func TestACardSaysWhoHasItAndWhenItMoved(t *testing.T) {
	c := read(doc.Parse([]byte("## Doing\n- table widget @codex-1 @{2026-10-03} @@{14:02}\n- mail a@b.com\n")))
	cd := c.cols[0].cards[0]
	if cd.text != "table widget" || strings.Join(cd.who, ",") != "codex-1" || !cd.moved.Equal(at(14, 2)) {
		t.Errorf("card = %+v", cd)
	}
	if c.cols[0].cards[1].text != "mail a@b.com" {
		t.Errorf("an address is not someone: %+v", c.cols[0].cards[1])
	}
}

// stickypane writes when a card moved or appeared, by itself: a card an
// agent moved by editing the file, a new card, a card without a time.
func TestTendWritesWhenACardMoved(t *testing.T) {
	before := doc.Parse([]byte("## To do\n- a @x @{2026-10-02} @@{09:00}\n- b @{2026-10-02} @@{09:00}\n## Doing\n"))
	after := doc.Parse([]byte("## To do\n- b @{2026-10-02} @@{09:00}\n- c\n## Doing\n- a @x @{2026-10-02} @@{09:00}\n"))
	op := Kind.Tend(before, after, at(14, 2))
	if op == nil {
		t.Fatal("a moved card needs a new time")
	}
	d, err := op.Apply(after)
	if err != nil {
		t.Fatal(err)
	}
	want := "## To do\n- b @{2026-10-02} @@{09:00}\n- c @{2026-10-03} @@{14:02}\n## Doing\n- a @x @{2026-10-03} @@{14:02}\n"
	if d.Body != want {
		t.Errorf("tended =\n%s\nwant\n%s", d.Body, want)
	}
	if Kind.Tend(d, d, at(15, 0)) != nil {
		t.Error("nothing moved, nothing to write")
	}
}

// Moving a card on the board takes its old time off, so it gets the new one.
func TestMovingACardTakesItsOldTimeOff(t *testing.T) {
	d, err := MoveCard{From: "To do", To: "Doing", Text: "a"}.Apply(doc.Parse([]byte("## To do\n- a @x @{2026-10-02} @@{09:00}\n## Doing\n")))
	if err != nil || d.Body != "## To do\n## Doing\n- a @x\n" {
		t.Errorf("moved = %q, %v", d.Body, err)
	}
}

// Under each card, who has it and when it moved; a card that has not moved
// for half an hour, outside the last column, is flagged.
func TestACardShowsWhoAndWhenAndWhetherItStalled(t *testing.T) {
	now = func() time.Time { return at(15, 0) }
	t.Cleanup(func() { now = func() time.Time { return widget.Now() } })
	b := parse(doc.Parse([]byte("## Doing\n- fresh @claude-a @{2026-10-03} @@{14:50}\n- stuck @codex-1 @{2026-10-03} @@{14:20}\n## Done\n- old @x @{2026-10-03} @@{09:00}\n")))
	out, _ := b.Draw(80, false)
	s := ansi.Strip(out)
	for _, want := range []string{"@claude-a · 14:50", "@codex-1 · 14:20 ⚠", "@x · 09:00"} {
		if !strings.Contains(s, want) {
			t.Errorf("want %q in\n%s", want, s)
		}
	}
	if strings.Contains(s, "09:00 ⚠") {
		t.Errorf("a card in the last column is done, not stalled:\n%s", s)
	}
	if next := b.NextChange(at(15, 0)); !next.Equal(at(15, 20)) {
		t.Errorf("the next card to stall does so at 15:20, got %v", next)
	}
}

// In a narrow column a long name is cut, never the time.
func TestALongNameIsCutBeforeTheTime(t *testing.T) {
	now = func() time.Time { return at(15, 0) }
	t.Cleanup(func() { now = func() time.Time { return widget.Now() } })
	b := parse(doc.Parse([]byte("## Doing\n- x @refactor/widget-modules @{2026-10-03} @@{14:55}\n## Done\n")))
	out, _ := b.Draw(40, false)
	if s := ansi.Strip(out); !strings.Contains(s, "· 14:55") || !strings.Contains(s, "@refa") {
		t.Errorf("the time stays whole:\n%s", s)
	}
}
