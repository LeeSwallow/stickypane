package checklist

import (
	"regexp"
	"strings"
	"time"

	"github.com/LeeSwallow/stickypane/internal/when"
)

// itemRe splits a checkbox line into: prefix up to "[", the mark, "] ", text.
var itemRe = regexp.MustCompile(`^(\s*[-*] \[)([ xX])(\] ?)(.*)$`)

// A finished item says when it was done at the end of its line:
//
//   - [x] write tests ✅ 2026-10-03 14:02
//
// It is the "✅ date" of Obsidian's Tasks, with the time added, so the file
// reads the same in other tools. A date alone is understood too.
const StampLayout = when.Stamp

var stampRe = regexp.MustCompile(`\s*✅\s*(\d{4}-\d{2}-\d{2})(?:[ T](\d{2}:\d{2}))?\s*$`)

// Stamp is the text written after a finished item.
func Stamp(t time.Time) string { return t.Format(StampLayout) }

// splitStamp returns an item's text without its done time, and the time
// when there is one.
func splitStamp(text string) (string, time.Time) {
	m := stampRe.FindStringSubmatchIndex(text)
	if m == nil {
		return text, time.Time{}
	}
	date := text[m[2]:m[3]]
	clock := "00:00"
	if m[4] >= 0 {
		clock = text[m[4]:m[5]]
	}
	at, ok := when.Parse(date + " " + clock)
	if !ok {
		return text, time.Time{}
	}
	return strings.TrimRight(text[:m[0]], " \t"), at
}

// marked writes an item's text in state checked: done at the time at, when
// at is given and the item is being ticked; without a time otherwise.
func marked(text string, checked bool, at string) string {
	text, _ = splitStamp(text)
	if checked && at != "" {
		return text + " ✅ " + at
	}
	return text
}

// hasStamp reports whether an item's text ends with its done time.
func hasStamp(text string) bool { return stampRe.MatchString(text) }
