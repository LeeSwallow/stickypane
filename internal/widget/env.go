package widget

import (
	"time"

	"charm.land/lipgloss/v2"

	"github.com/LeeSwallow/stickypane/internal/theme"
	"github.com/LeeSwallow/stickypane/internal/when"
)

// StaleAfter is how long a card may stay where it is, outside the last
// column, before a board flags it, unless the settings say otherwise.
const StaleAfter = 30 * time.Minute

// Env is what a widget draws and acts with that is not in its file: the
// clock, the words of the user's language, the colors of the theme, how
// long a card may stall and how a time is written. The screen makes one
// from its own clock and settings and hands it to every widget it parses;
// the command line and the bridge make theirs from the API's clock. Nothing a widget reads is a
// package variable, so two boards, or two tests, never share one.
//
// The zero Env reads time.Now, keeps phrases as written, draws without
// color, never flags a card and writes times in the neutral form.
type Env struct {
	Clock      func() time.Time           // the time it is now; nil is time.Now
	Translate  func(phrase string) string // a phrase in the user's language; nil keeps it
	Styles     theme.Styles               // the text styles of the theme in use
	StaleAfter time.Duration              // how long a card may stay put; zero never flags
	Times      when.Formatter             // how the screen writes a time: the user's country's way
	// Embed finds the note a ![[name]] line shows: its title and its
	// widget, drawn read only. Nil draws an embed as a link.
	Embed func(name string) (title string, w Widget, ok bool)
	// User is the name the user writes as (env.Detect().User()), asked
	// when a widget says something as the user. Nil is no name.
	User func() string
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

// Who is the name the user writes as, by the Env's User.
func (e Env) Who() string {
	if e.User == nil {
		return ""
	}
	return e.User()
}

// T translates a phrase a widget shows.
func (e Env) T(phrase string) string {
	if e.Translate == nil {
		return phrase
	}
	return e.Translate(phrase)
}

// Short says when t was, as the Env's Formatter writes it next to the
// Env's clock: the time of day for today, the day for this year, the date
// before that.
func (e Env) Short(t time.Time) string { return e.Times.Short(t, e.Now()) }

// NameStyle gives a name (an @owner, a chat's author) its color, the same
// each time it appears and in every note.
func (e Env) NameStyle(name string) lipgloss.Style { return nameStyle(e.Styles, name) }
