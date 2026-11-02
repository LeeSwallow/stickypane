package widget

import (
	"fmt"
	"strings"
)

// The board can be set to a fixed number of columns (the "columns"
// setting). Then a note without a size of its own takes one column, and a
// note with one takes the share of the columns its fraction comes to. The
// rule is here once; the desktop app's board/layout.ts columnSpan reads
// the same table, testdata/columns.json, in its tests.

// MaxColumns is the most columns a board can be set to.
const MaxColumns = 4

// ColumnSpan is how many of n columns a note of size s takes: its fraction of n,
// rounded half up, at least one and at most n. A card, or a size that is
// not one, takes one column. Every k/n it can come to is one of Sizes, so
// stepping by a column writes a size the board already reads.
func ColumnSpan(s string, n int) int {
	if n <= 1 {
		return 1
	}
	s, ok := NormalSize(s)
	if !ok || s == SizeCard {
		return 1
	}
	num, den := 1, 1
	if i := strings.IndexByte(s, '/'); i > 0 {
		num, den = int(s[0]-'0'), int(s[i+1]-'0')
	}
	return max(min((2*num*n+den)/(2*den), n), 1)
}

// ColumnSize is the size k of n columns is written as: one of Sizes.
func ColumnSize(k, n int) string {
	n = max(n, 1)
	k = max(min(k, n), 1)
	if k == n {
		return SizeFull
	}
	a, b := k, n
	for b != 0 {
		a, b = b, a%b
	}
	return fmt.Sprintf("%d/%d", k/a, n/a)
}

// StepColumns is the size one column wider (delta 1) or narrower (-1)
// than a note of size s on n columns, staying within one and n.
func StepColumns(s string, n, delta int) string {
	return ColumnSize(ColumnSpan(s, n)+delta, n)
}

// SnapColumns is the size a dragged border lands on, on n columns across
// a screen width cells wide: the nearest whole number of columns.
func SnapColumns(cells, width, n int) string {
	n, width = max(n, 1), max(width, 1)
	return ColumnSize((2*cells*n+width)/(2*width), n)
}

// FitColumns is how many columns of the n chosen a screen width cells wide draws:
// n, or fewer when a column would be narrower than MinWidth, and never
// fewer than one. It is a safeguard for a small screen, not a layout of
// its own.
func FitColumns(n, width int) int {
	return max(min(n, width/MinWidth), 1)
}
