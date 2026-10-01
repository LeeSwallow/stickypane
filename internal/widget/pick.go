package widget

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

// ErrNoMatch reports that Pick found nothing by the name it was given.
var ErrNoMatch = errors.New("no match")

// listed is how many names an error message spells out.
const listed = 12

// Pick finds which of names is meant by want, for commands that name an
// item, a card or a column instead of pointing at it. In order: the exact
// text; a position written "#3" or "3", counting from one; the same text in
// another case; and a part of the text, when only one name has it. what
// says what the names are ("item", "card") for the error message, which
// lists the names so the caller can try again without reading the note.
func Pick(names []string, want, what string) (int, error) {
	want = strings.TrimSpace(want)
	if len(names) == 0 {
		return -1, fmt.Errorf("%w: there is no %s yet", ErrNoMatch, what)
	}
	for i, n := range names {
		if n == want {
			return i, nil
		}
	}
	if n, err := strconv.Atoi(strings.TrimPrefix(want, "#")); err == nil {
		if n < 1 || n > len(names) {
			return -1, fmt.Errorf("%w: there is no %s %d, only %d", ErrNoMatch, what, n, len(names))
		}
		return n - 1, nil
	}
	var same, part []int
	for i, n := range names {
		switch {
		case strings.EqualFold(n, want):
			same = append(same, i)
		case want != "" && strings.Contains(strings.ToLower(n), strings.ToLower(want)):
			part = append(part, i)
		}
	}
	switch {
	case len(same) > 0:
		return same[0], nil
	case len(part) == 1:
		return part[0], nil
	case len(part) > 1:
		return -1, fmt.Errorf("%q could be more than one %s: %s", want, what, quoted(names, part))
	}
	all := make([]int, len(names))
	for i := range names {
		all[i] = i
	}
	return -1, fmt.Errorf("%w: no %s is called %q; there %s", ErrNoMatch, what, want, isAre(len(names), quoted(names, all)))
}

func isAre(n int, list string) string {
	if n == 1 {
		return "is " + list
	}
	return "are " + list
}

func quoted(names []string, which []int) string {
	var parts []string
	for _, i := range which {
		if len(parts) == listed {
			parts = append(parts, fmt.Sprintf("and %d more", len(which)-listed))
			break
		}
		parts = append(parts, strconv.Quote(names[i]))
	}
	return strings.Join(parts, ", ")
}
