// Package when writes and reads the times the board shows: when an item was
// done, when a form was sent, when a log last grew, how long a script ran.
// Every widget says a time the same way because they all ask here.
//
// Times on the screen are absolute ("14:02", "Sep 30"), never relative
// ("3m ago"): the board redraws when a file changes, not as the clock
// moves, and a relative time would quietly go stale.
package when

import (
	"fmt"
	"strings"
	"time"
)

// The layouts written into files.
const (
	Stamp = "2006-01-02 15:04" // a moment in a note: "- [x] tests ✅ 2026-10-03 14:02"
	Clock = "15:04"            // a time of day: "14:02 tests passed" in a log
	Date  = "2006-01-02"
	// Moment is a moment in a record kept for machines: "2026-10-28T14:02:11+09:00".
	Moment = time.RFC3339
)

// The layouts of lines the board writes for people to read beside a run.
const (
	started = "2006-01-02 15:04:05" // the head of a script's log
	second  = "15:04:05"            // a line of stickypane watch
	milli   = "15:04:05.000"        // a frame of a WebSocket transcript
	compact = "20060102-150405"     // a file name made from a time
)

// LogHead writes when a script started, in the head of its log:
// "2026-10-03 14:02:05".
func LogHead(t time.Time) string { return t.Format(started) }

// Second writes the time of day to the second, as a line of stickypane
// watch starts: "14:02:05".
func Second(t time.Time) string { return t.Format(second) }

// Milli writes the time of day to the millisecond, as a line of a
// WebSocket transcript says it: "14:02:05.042".
func Milli(t time.Time) string { return t.Format(milli) }

// Slug writes a time as part of a file name, without spaces or colons:
// "20261003-140205".
func Slug(t time.Time) string { return t.Format(compact) }

// Formatter writes times on the screen the way a country does: a time of
// day, a day of this year and a day of another year, as Go layouts. What is
// written into files stays in the layouts above, which every tool reads the
// same. The zero Formatter writes a neutral form: 14:02, Jan 2, 2025-12-31.
type Formatter struct{ clock, day, year string }

// neutralDay is how the zero Formatter writes a day of this year.
const neutralDay = "Jan 2"

// countries are the ways of writing times, by language and country. The
// first entry whose language and, when it names one, country match the
// locale wins.
var countries = []struct {
	lang, country       string
	clock, day, yearDay string
}{
	{"ko", "", Clock, "1월 2일", "2006. 1. 2."},
	{"ja", "", Clock, "1月2日", "2006/01/02"},
	{"zh", "", Clock, "1月2日", "2006/01/02"},
	{"en", "US", "3:04 PM", "Jan 2", "Jan 2, 2006"},
	{"en", "CA", "3:04 PM", "Jan 2", "Jan 2, 2006"},
	{"en", "PH", "3:04 PM", "Jan 2", "Jan 2, 2006"},
	{"en", "", Clock, "2 Jan", "2 Jan 2006"},
	{"de", "", Clock, "02.01.", "02.01.2006"},
	{"ru", "", Clock, "02.01.", "02.01.2006"},
	{"pl", "", Clock, "02.01.", "02.01.2006"},
	{"fr", "", Clock, "02/01", "02/01/2006"},
	{"es", "", Clock, "02/01", "02/01/2006"},
	{"it", "", Clock, "02/01", "02/01/2006"},
	{"pt", "", Clock, "02/01", "02/01/2006"},
	{"nl", "", Clock, "02-01", "02-01-2006"},
}

// Locale returns the Formatter that writes times the way the country of
// locale does ("ko_KR.UTF-8", "en-GB"). An empty locale, "C" or "POSIX", or
// one not known, gives the zero Formatter: 14:02, Jan 2, 2025-12-31.
func Locale(locale string) Formatter {
	lang, country := splitLocale(locale)
	if lang == "" {
		return Formatter{}
	}
	for _, c := range countries {
		if c.lang == lang && (c.country == "" || c.country == country) {
			return Formatter{c.clock, c.day, c.yearDay}
		}
	}
	return Formatter{}
}

func (f Formatter) layouts() (clock, day, year string) {
	if f == (Formatter{}) {
		return Clock, neutralDay, Date
	}
	return f.clock, f.day, f.year
}

// splitLocale turns "en_US.UTF-8" or "en-US" into "en" and "US". An
// English locale without a country is American, as the C library takes it.
func splitLocale(locale string) (lang, country string) {
	if i := strings.IndexAny(locale, ".@"); i >= 0 {
		locale = locale[:i]
	}
	lang, country, _ = strings.Cut(strings.ReplaceAll(locale, "_", "-"), "-")
	lang, country = strings.ToLower(lang), strings.ToUpper(country)
	if lang == "c" || lang == "posix" {
		return "", ""
	}
	if lang == "en" && country == "" {
		country = "US"
	}
	return lang, country
}

// Short says when t was in as few characters as the day allows: the time of
// day for today, the day for this year, the date before that. The zero time
// says nothing.
func (f Formatter) Short(t, now time.Time) string {
	if t.IsZero() {
		return ""
	}
	clock, day, year := f.layouts()
	t, now = t.Local(), now.Local()
	switch {
	case t.Year() == now.Year() && t.YearDay() == now.YearDay():
		return t.Format(clock)
	case t.Year() == now.Year():
		return t.Format(day)
	}
	return t.Format(year)
}

// Duration says how long something took, to the precision that matters at
// its length: "0.42s", "12.3s", "2m 03s", "1h 02m".
func Duration(d time.Duration) string {
	switch {
	case d <= 0:
		return "0s"
	case d < 10*time.Second:
		return trim(fmt.Sprintf("%.2f", d.Seconds())) + "s"
	case d < time.Minute:
		return trim(fmt.Sprintf("%.1f", d.Seconds())) + "s"
	case d < time.Hour:
		return fmt.Sprintf("%dm %02ds", int(d.Minutes()), int(d.Seconds())%60)
	}
	return fmt.Sprintf("%dh %02dm", int(d.Hours()), int(d.Minutes())%60)
}

func trim(s string) string {
	if strings.Contains(s, ".") {
		s = strings.TrimRight(strings.TrimRight(s, "0"), ".")
	}
	return s
}

// Parse reads a time as the files write it: RFC 3339 (a form's
// submitted_at), Stamp, or a date alone. A time without a zone is local.
func Parse(s string) (time.Time, bool) {
	s = strings.TrimSpace(s)
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t, true
	}
	for _, layout := range []string{Stamp, "2006-01-02T15:04", Date} {
		if t, err := time.ParseInLocation(layout, s, time.Local); err == nil {
			return t, true
		}
	}
	return time.Time{}, false
}
