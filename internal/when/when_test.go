package when

import (
	"testing"
	"time"
)

var now = time.Date(2026, 10, 3, 18, 0, 0, 0, time.Local)

func TestShortSaysAsLittleAsTheDayAllows(t *testing.T) {
	for _, c := range []struct {
		t    time.Time
		want string
	}{
		{time.Date(2026, 10, 3, 14, 2, 0, 0, time.Local), "14:02"},
		{time.Date(2026, 9, 30, 10, 0, 0, 0, time.Local), "Sep 30"},
		{time.Date(2025, 12, 31, 10, 0, 0, 0, time.Local), "2025-12-31"},
		{time.Time{}, ""},
	} {
		if got := Short(c.t, now); got != c.want {
			t.Errorf("Short(%v) = %q, want %q", c.t, got, c.want)
		}
	}
	old := DayLayout
	DayLayout = "1월 2일"
	t.Cleanup(func() { DayLayout = old })
	if got := Short(time.Date(2026, 9, 30, 10, 0, 0, 0, time.Local), now); got != "9월 30일" {
		t.Errorf("a translated day = %q", got)
	}
}

func TestDurationIsReadable(t *testing.T) {
	for d, want := range map[time.Duration]string{
		420 * time.Millisecond:                     "0.42s",
		1023 * time.Millisecond:                    "1.02s",
		12345 * time.Millisecond:                   "12.3s",
		2*time.Minute + 3*time.Second:              "2m 03s",
		time.Hour + 2*time.Minute + 59*time.Second: "1h 02m",
		0: "0s",
	} {
		if got := Duration(d); got != want {
			t.Errorf("Duration(%v) = %q, want %q", d, got, want)
		}
	}
}

func TestParseReadsTheFormsTheFilesUse(t *testing.T) {
	for s, want := range map[string]string{
		"2026-10-03 14:02": "2026-10-03 14:02",
		"2026-10-03":       "2026-10-03 00:00",
	} {
		got, ok := Parse(s)
		if !ok || got.Format(Stamp) != want {
			t.Errorf("Parse(%q) = %v, %v", s, got.Format(Stamp), ok)
		}
	}
	if got, ok := Parse("2026-10-03T14:02:05+09:00"); !ok || !got.Equal(time.Date(2026, 10, 3, 5, 2, 5, 0, time.UTC)) {
		t.Errorf("an RFC 3339 time is the same moment: %v", got)
	}
	if _, ok := Parse("soon"); ok {
		t.Error("not a time")
	}
}

// The screen writes days and times the way the user's country does, as the
// locale of the environment names it.
func TestTimesFollowTheCountry(t *testing.T) {
	t.Cleanup(func() { UseLocale("") })
	today := time.Date(2026, 10, 3, 14, 2, 0, 0, time.Local)
	earlier := time.Date(2026, 9, 30, 9, 5, 0, 0, time.Local)
	lastYear := time.Date(2025, 12, 31, 9, 5, 0, 0, time.Local)
	for _, c := range []struct{ locale, today, earlier, lastYear string }{
		{"ko_KR.UTF-8", "14:02", "9월 30일", "2025. 12. 31."},
		{"en_US.UTF-8", "2:02 PM", "Sep 30", "Dec 31, 2025"},
		{"en-GB", "14:02", "30 Sep", "31 Dec 2025"},
		{"ja_JP", "14:02", "9月30日", "2025/12/31"},
		{"de_DE.UTF-8", "14:02", "30.09.", "31.12.2025"},
		{"fr_FR", "14:02", "30/09", "31/12/2025"},
		{"", "14:02", "Sep 30", "2025-12-31"},
		{"C", "14:02", "Sep 30", "2025-12-31"},
	} {
		UseLocale(c.locale)
		got := []string{Short(today, now), Short(earlier, now), Short(lastYear, now)}
		want := []string{c.today, c.earlier, c.lastYear}
		for i := range got {
			if got[i] != want[i] {
				t.Errorf("%q: %q, want %q", c.locale, got[i], want[i])
			}
		}
	}
}
