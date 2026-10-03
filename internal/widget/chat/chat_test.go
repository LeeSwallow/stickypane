package chat

import (
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/x/ansi"

	"github.com/LeeSwallow/stickypane/internal/doc"
	"github.com/LeeSwallow/stickypane/internal/widget"
)

const src = "---\ntype: chat\n---\n## 2026-10-03\n@min 14:02 can you rerun the tests?\n@claude 14:03 41/41 pass\n  see test.log\n"

func TestAChatReadsItsMessages(t *testing.T) {
	c := read(doc.Parse([]byte(src)))
	if len(c) != 2 || c[0].who != "min" || c[1].text != "41/41 pass\nsee test.log" || c[1].day != "2026-10-03" {
		t.Errorf("messages = %+v", c)
	}
	out, _ := parse(doc.Parse([]byte(src))).Draw(60, false)
	s := ansi.Strip(out)
	for _, want := range []string{"2026-10-03", "@min", "14:02", "can you rerun the tests?", "@claude", "see test.log"} {
		if !strings.Contains(s, want) {
			t.Errorf("want %q in\n%s", want, s)
		}
	}
	if got := parse(doc.Parse([]byte(src))).Summary(); got != "2 messages" {
		t.Errorf("Summary = %q", got)
	}
}

// Saying something adds it under today's heading, with the time.
func TestSayAddsAMessageUnderToday(t *testing.T) {
	at := time.Date(2026, 10, 4, 9, 30, 0, 0, time.Local)
	d, err := Say{Who: "min", Text: "good morning\nagain", At: at}.Apply(doc.Parse([]byte(src)))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasSuffix(d.Body, "  see test.log\n\n## 2026-10-04\n@min 09:30 good morning again\n") {
		t.Errorf("body = %q", d.Body)
	}
	d, _ = Say{Who: "claude", Text: "hi", At: at}.Apply(d)
	if !strings.HasSuffix(d.Body, "## 2026-10-04\n@min 09:30 good morning again\n@claude 09:30 hi\n") {
		t.Errorf("a second message the same day goes under the same heading: %q", d.Body)
	}
}

func TestEventsSayWhoSaidWhat(t *testing.T) {
	before := doc.Parse([]byte(src))
	after, _ := Say{Who: "min", Text: "thanks", At: time.Date(2026, 10, 3, 14, 5, 0, 0, time.Local)}.Apply(before)
	evs := events(before, after)
	if len(evs) != 1 || evs[0].Type != "message.added" || evs[0].Item != "thanks" || evs[0].From != "min" {
		t.Errorf("events = %+v", evs)
	}
	var _ widget.Event
}
