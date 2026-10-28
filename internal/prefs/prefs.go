// Package prefs is the list of what a user may change about the board, each
// with a default, so that nothing has to be set: the board works as it is,
// and the settings panel (S) and `stickypane config` change what someone
// wants different. The values live in the root sticky.json.
package prefs

import (
	"fmt"
	"slices"
	"strings"

	"github.com/LeeSwallow/stickypane/internal/env"
	"github.com/LeeSwallow/stickypane/internal/i18n"
	"github.com/LeeSwallow/stickypane/internal/store"
	"github.com/LeeSwallow/stickypane/internal/theme"
)

// Pref is one setting.
type Pref struct {
	Key     string          // the key in sticky.json and for stickypane config
	Label   string          // the name on the settings panel
	Help    string          // what it changes, in a line
	Default string          // the value when none is set: the first of Values
	Values  func() []string // every value it may take
}

// All are the settings, in the panel's order.
var All = []Pref{
	{
		Key: "theme", Label: "Theme", Default: "auto",
		Help:   "the colors; auto follows the terminal's background",
		Values: func() []string { return append([]string{"auto"}, names(theme.All())...) },
	},
	{
		Key: "language", Label: "Language", Default: "auto",
		Help:   "the screen's words; auto follows STICKYPANE_LANG, LANG and the system",
		Values: func() []string { return append([]string{"auto"}, i18n.Languages()...) },
	},
	{
		Key: "time", Label: "Time format", Default: "auto",
		Help:   "how days and times are written; auto follows the country of the locale",
		Values: func() []string { return []string{"auto", "ko", "en-US", "en-GB", "ja", "de", "fr", "iso"} },
	},
	{
		Key: "editor", Label: "Editor", Default: "auto",
		Help:   "what E opens; auto is $VISUAL, $EDITOR, then the first editor found",
		Values: func() []string { return append([]string{"auto"}, env.Detect().InstalledEditors()...) },
	},
	{
		Key: "stale", Label: "Cards stall after", Default: "30m",
		Help:   "when a kanban card that has not moved is flagged ⚠",
		Values: func() []string { return []string{"30m", "15m", "1h", "4h", "1d", "off"} },
	},
}

func names(ts []theme.Theme) []string {
	out := make([]string, len(ts))
	for i, t := range ts {
		out[i] = t.Name
	}
	return out
}

// Lookup finds a setting by key; the zero Pref when there is none.
func Lookup(key string) Pref {
	for _, p := range All {
		if p.Key == key {
			return p
		}
	}
	return Pref{}
}

// Get is a setting's value: what sticky.json says, else its default.
func Get(set store.Settings, key string) string {
	if v := set.Root(key); v != "" {
		return v
	}
	p, _ := find(key)
	return p.Default
}

// Screen are the settings of how the board is shown, which the desktop app
// changes and the terminal's panel does not list: the view and the style.
var Screen = []Pref{
	{
		Key: "view", Label: "View", Default: store.ViewTerminal,
		Help:   "where the board is shown: the terminal, the desktop overlay, or both",
		Values: func() []string { return []string{store.ViewTerminal, store.ViewOverlay, store.ViewBoth} },
	},
	{
		Key: "style", Label: "Style", Default: store.StyleDoodle,
		Help:   "how the overlay draws its notes",
		Values: func() []string { return []string{store.StyleDoodle, store.StyleNeat} },
	},
}

// find is a setting of the panel or of the screen by key.
func find(key string) (Pref, bool) {
	for _, p := range append(slices.Clone(All), Screen...) {
		if p.Key == key {
			return p, true
		}
	}
	return Pref{}, false
}

// Refused is a value a setting cannot take. What it says is the one
// sentence every front end shows; a front end that lists the values
// itself reads Values.
type Refused struct {
	Key, Value string
	Values     []string // what the setting takes
}

func (r *Refused) Error() string {
	switch r.Key {
	case "theme":
		return fmt.Sprintf("there is no theme %q", r.Value)
	case "language":
		return fmt.Sprintf("there is no translation for %q: use %s", r.Value, strings.Join(r.Values, ", "))
	case "style":
		return fmt.Sprintf("style %q is not doodle or neat", r.Value)
	case "view":
		return fmt.Sprintf("view %q is not terminal, overlay or both", r.Value)
	}
	return fmt.Sprintf("%s is one of %s, not %q", r.Key, strings.Join(r.Values, ", "), r.Value)
}

// Normal is the value a setting keeps for what was asked: a theme by its
// name in any spelling theme.Lookup reads, a language code in lower case,
// and any other value as its list writes it. A value the setting cannot
// take is a *Refused; a setting that does not exist is an error that lists
// the settings.
func Normal(key, value string) (string, error) {
	p, ok := find(key)
	if !ok {
		keys := make([]string, len(All))
		for i, q := range All {
			keys[i] = q.Key
		}
		return "", fmt.Errorf("there is no setting %q; there are %s", key, strings.Join(keys, ", "))
	}
	vals := p.Values()
	switch v := strings.TrimSpace(value); {
	case strings.EqualFold(v, "auto") && (key == "theme" || key == "language"):
		return "auto", nil
	case key == "theme":
		if t, ok := theme.Lookup(v); ok {
			return t.Name, nil
		}
	case key == "language":
		code := strings.ToLower(v)
		if _, err := i18n.Load(code); err == nil && code != "" {
			return code, nil
		}
	default:
		if i := slices.IndexFunc(vals, func(s string) bool { return strings.EqualFold(s, v) }); i >= 0 {
			return vals[i], nil
		}
	}
	return "", &Refused{Key: key, Value: value, Values: vals}
}

// Change is one setting given a value.
type Change struct{ Key, Value string }

// Apply checks every change and then writes them, so a value a setting
// cannot take changes nothing. A value at its setting's default is
// written as no value at all.
func Apply(st *store.Store, changes ...Change) error {
	keep := make([]Change, len(changes))
	for i, c := range changes {
		v, err := Normal(c.Key, c.Value)
		if err != nil {
			return err
		}
		if p, _ := find(c.Key); v == p.Default {
			v = ""
		}
		keep[i] = Change{c.Key, v}
	}
	for _, c := range keep {
		if err := st.SetRoot(c.Key, c.Value); err != nil {
			return err
		}
	}
	return nil
}

// Set changes one setting, as Apply does.
func Set(st *store.Store, key, value string) error { return Apply(st, Change{key, value}) }

// Next is the value after (or, with step -1, before) v, going round.
func (p Pref) Next(v string, step int) string {
	vals := p.Values()
	i := slices.Index(vals, v)
	if i < 0 {
		return vals[0]
	}
	return vals[((i+step)%len(vals)+len(vals))%len(vals)]
}
