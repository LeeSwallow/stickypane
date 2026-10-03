package chart

import (
	"regexp"
	"strconv"
	"strings"
)

// dataRe matches a data line: an optional list marker, a label, a colon,
// a space and a number that may use "," or "_" between digits. The space is
// required, so that a time ("12:30") or an address ("host:8080") in a line
// of text is not taken for data.
var dataRe = regexp.MustCompile(`^\s*(?:[-*]\s+)?(.+?)\s*:\s+(-?[0-9][0-9,_]*(?:\.[0-9]+)?)\s*$`)

// numberRe matches a value as a data line writes it.
var numberRe = regexp.MustCompile(`^-?[0-9][0-9,_]*(\.[0-9]+)?$`)

// number reads a value written with "," or "_" between its digits.
func number(text string) (float64, error) {
	return strconv.ParseFloat(strings.NewReplacer(",", "", "_", "").Replace(text), 64)
}

// find returns the line of the value called label, and where its number is
// on that line. Labels are compared without regard to case.
func find(lines []string, label string) (line, from, to int) {
	for i, l := range lines {
		raw := strings.TrimSuffix(l, "\r")
		if m := dataRe.FindStringSubmatchIndex(raw); m != nil && strings.EqualFold(raw[m[2]:m[3]], label) {
			return i, m[4], m[5]
		}
	}
	return -1, 0, 0
}
