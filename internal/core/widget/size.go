package widget

import "strings"

// Sizes are the widths an open note can take, as fractions of the screen,
// narrowest first: the steps "+" and "-" go through, and what sticky.json
// keeps as "size".
var Sizes = []string{"1/4", "1/3", "1/2", "2/3", "3/4", "1"}

const (
	SizeFull = "1"   // the whole width
	SizeHalf = "1/2" // half the width
	// SizeCard is a small sticky note: the narrowest step a card of
	// CardWidth cells fits in, so it depends on the screen. It is also
	// what the old name "card" reads as.
	SizeCard = "card"
)

// CardWidth is the narrowest a card is drawn when the screen allows it.
const CardWidth = 36

// MinWidth is the narrowest a note is drawn: a fraction that would be
// narrower on a small screen takes the next one that is not, or the whole
// width.
const MinWidth = 20

// aliases are the names sizes had before they were fractions. They are
// still read; the board writes a fraction the next time the size changes.
var aliases = map[string]string{"page": SizeFull, "half": SizeHalf, SizeCard: SizeCard}

// NormalSize returns a size as the board writes it: one of Sizes, or
// SizeCard. ok is false when s is not a size.
func NormalSize(s string) (string, bool) {
	s = strings.ToLower(strings.TrimSpace(s))
	if a, ok := aliases[s]; ok {
		return a, true
	}
	for _, f := range Sizes {
		if s == f {
			return f, true
		}
	}
	return "", false
}

// SizeCells is how many cells wide a note of size s is on a screen width
// cells wide: its share, but never under MinWidth when a wider step is
// left. A size that is not one is the whole width.
func SizeCells(s string, width int) int {
	for i := step(s, width); i < len(Sizes); i++ {
		if cells := share(Sizes[i], width); cells >= MinWidth || i == len(Sizes)-1 {
			return max(cells, 1)
		}
	}
	return max(width, 1)
}

// share is a fraction of width, rounded down.
func share(f string, width int) int {
	num, den := 1, 1
	if i := strings.IndexByte(f, '/'); i > 0 {
		num, den = int(f[0]-'0'), int(f[i+1]-'0')
	}
	return width * num / den
}

// cardStep is the step a card takes on a screen width cells wide.
func cardStep(width int) string {
	for _, s := range Sizes {
		if share(s, width) >= CardWidth {
			return s
		}
	}
	return SizeFull
}

// step returns where a size is among Sizes on a screen width cells wide;
// a size that is not one is the whole width.
func step(s string, width int) int {
	s, _ = NormalSize(s)
	if s == SizeCard {
		s = cardStep(width)
	}
	for i, f := range Sizes {
		if f == s {
			return i
		}
	}
	return len(Sizes) - 1
}

// StepSize returns the size delta steps from s, staying within Sizes. A
// card counts as the step it takes on a screen width cells wide.
func StepSize(s string, width, delta int) string {
	return Sizes[max(min(step(s, width)+delta, len(Sizes)-1), 0)]
}

// SnapSize returns the step a dragged border lands on: the step from was
// at, until cells passes the middle between it and the next step either
// way, so a border moved a cell or two does not change the size. cells is
// the width the note asks for, not the width a stretched row gives it.
func SnapSize(from string, cells, width int) string {
	i := step(from, width)
	mid := func(a, b int) int { return SizeCells(Sizes[a], width) + SizeCells(Sizes[b], width) }
	for i < len(Sizes)-1 && 2*cells > mid(i, i+1) {
		i++
	}
	for i > 0 && 2*cells < mid(i-1, i) {
		i--
	}
	return Sizes[i]
}
