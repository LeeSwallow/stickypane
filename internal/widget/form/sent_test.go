package form

import (
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/x/ansi"

	"github.com/LeeSwallow/stickypane/internal/doc"
	"github.com/LeeSwallow/stickypane/internal/widget/note"
)

// A form sent on another day says the day, not only a time of day.
func TestASentFormSaysWhen(t *testing.T) {
	now = func() time.Time { return time.Date(2026, 10, 3, 18, 0, 0, 0, time.Local) }
	t.Cleanup(func() { now = time.Now })
	for at, want := range map[string]string{
		time.Date(2026, 10, 3, 14, 2, 0, 0, time.Local).Format(time.RFC3339): "sent 14:02",
		time.Date(2026, 9, 30, 9, 0, 0, 0, time.Local).Format(time.RFC3339):  "sent Sep 30",
	} {
		f := parse(doc.Parse([]byte("---\ntype: form\nsubmitted: OK\nsubmitted_at: "+at+"\n---\n- ( ) a\n")), note.Plain)
		out, _ := f.Draw(40, false)
		if !strings.Contains(ansi.Strip(out), want) {
			t.Errorf("submitted_at %s: want %q in\n%s", at, want, ansi.Strip(out))
		}
	}
}

// A sent form shows what it produced: the answers, as stickypane answers
// prints them, under a rule with the button and when.
func TestASentFormShowsItsAnswers(t *testing.T) {
	now = func() time.Time { return time.Date(2026, 10, 3, 18, 0, 0, 0, time.Local) }
	t.Cleanup(func() { now = time.Now })
	src := "---\ntype: form\nsubmitted: Deploy\nsubmitted_at: " + time.Date(2026, 10, 3, 14, 2, 0, 0, time.Local).Format(time.RFC3339) +
		"\n---\n## Target\n- ( ) staging\n- (x) production\n\n## Also\n- [x] run migrations\n- [ ] clear the cache\n\n## Note\n> after lunch\n\n[ Deploy ] [ Cancel ]\n"
	out, _ := parse(doc.Parse([]byte(src)), note.Plain).Draw(50, false)
	s := ansi.Strip(out)
	for _, want := range []string{"Deploy · sent 14:02", "Target: production", "Also: run migrations", "Note: after lunch"} {
		if !strings.Contains(s, want) {
			t.Errorf("want %q in\n%s", want, s)
		}
	}
	if before, _ := parse(doc.Parse([]byte(strings.Replace(src, "submitted: Deploy\n", "", 1))), note.Plain).Draw(50, false); strings.Contains(ansi.Strip(before), "Target: production") {
		t.Error("a form not sent yet has no answers to show")
	}
}
