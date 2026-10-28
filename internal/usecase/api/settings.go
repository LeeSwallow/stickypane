package api

import (
	"github.com/LeeSwallow/stickypane/internal/adapter/store"
	"github.com/LeeSwallow/stickypane/internal/usecase/prefs"
)

// Setting is a change to one of the board's settings: the theme, the
// language, how times are written, the editor, when a card stalls, the
// view and the style.
type Setting = prefs.Change

// ChangeSettings is the one way a front end changes the board's settings:
// the terminal's panel, the bridge's settings.set, and the theme, language
// and config commands. Every change is checked (prefs.Normal) before any
// is written, so a value a setting cannot take, a *prefs.Refused, changes
// nothing.
func (a *API) ChangeSettings(changes ...Setting) error { return prefs.Apply(a.st, changes...) }

// Settings is what the board's settings say now, read in one pass.
func (a *API) Settings() store.Settings { return a.st.Settings() }
