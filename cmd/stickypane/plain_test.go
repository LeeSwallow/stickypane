package main

import "regexp"

// The times stickypane writes by itself: when a note was made, and when an
// item was done.
var (
	createdLine = regexp.MustCompile(`(?m)^created: [0-9 :T+-]+\r?\n`)
	doneStamp   = regexp.MustCompile(` ✅ \d{4}-\d{2}-\d{2}( \d{2}:\d{2})?`)
	emptyFront  = regexp.MustCompile(`^---\r?\n---\r?\n`)
	movedStamp  = regexp.MustCompile(` @\{\d{4}-\d{2}-\d{2}\}( @@\{\d{2}:\d{2}\})?`)
)

// plain is a note's text without the times stickypane keeps in it, for
// tests about something else.
func plain(s string) string {
	s = doneStamp.ReplaceAllString(createdLine.ReplaceAllString(s, ""), "")
	s = movedStamp.ReplaceAllString(s, "")
	return emptyFront.ReplaceAllString(s, "")
}
