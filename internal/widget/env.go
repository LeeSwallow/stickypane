package widget

import (
	"time"

	"charm.land/lipgloss/v2"

	"github.com/LeeSwallow/stickypane/internal/theme"
)

// StaleAfter is how long a card may stay where it is, outside the last
// column, before a board flags it, unless the settings say otherwise.
const StaleAfter = 30 * time.Minute

// Env is what a widget draws and acts with that is not in its file: the
// clock, the words of the user's language, the colors of the theme and how
// long a card may stall. The screen makes one from its own clock and
// settings and hands it to every widget it parses; the command line and
// the bridge make theirs from the API's clock. Nothing a widget reads is a
// package variable, so two boards, or two tests, never share one.
//
// The zero Env reads time.Now, keeps phrases as written, draws without
// color and never flags a card.
type Env struct {
	Clock      func() time.Time           // the time it is now; nil is time.Now
	Translate  func(phrase string) string // a phrase in the user's language; nil keeps it
	Styles     theme.Styles               // the text styles of the theme in use
	StaleAfter time.Duration              // how long a card may stay put; zero never flags
}

// DefaultEnv is the Env of a screen that has not been told otherwise: the
// system clock, English, the default dark theme and cards that stall after
// StaleAfter.
func DefaultEnv() Env {
	return Env{Clock: time.Now, Styles: theme.Pick("auto", true).Styles(), StaleAfter: StaleAfter}
}

// Now is the time it is, by the Env's clock.
func (e Env) Now() time.Time {
	if e.Clock == nil {
		return time.Now()
	}
	return e.Clock()
}

// T translates a phrase a widget shows.
func (e Env) T(phrase string) string {
	if e.Translate == nil {
		return phrase
	}
	return e.Translate(phrase)
}

// NameStyle gives a name (an @owner, a chat's author) its color, the same
// each time it appears and in every note.
func (e Env) NameStyle(name string) lipgloss.Style { return nameStyle(e.Styles, name) }
