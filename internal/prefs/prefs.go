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
	return Lookup(key).Default
}

// Set changes a setting. A value it cannot take, or a setting that does
// not exist, is an error that lists what there is. The default is written
// as no value at all.
func Set(st *store.Store, key, value string) error {
	p := Lookup(key)
	if p.Key == "" {
		keys := make([]string, len(All))
		for i, q := range All {
			keys[i] = q.Key
		}
		return fmt.Errorf("there is no setting %q; there are %s", key, strings.Join(keys, ", "))
	}
	vals := p.Values()
	i := slices.IndexFunc(vals, func(v string) bool { return strings.EqualFold(v, value) })
	if i < 0 {
		return fmt.Errorf("%s is one of %s, not %q", key, strings.Join(vals, ", "), value)
	}
	if vals[i] == p.Default {
		return st.SetRoot(key, "")
	}
	return st.SetRoot(key, vals[i])
}

// Next is the value after (or, with step -1, before) v, going round.
func (p Pref) Next(v string, step int) string {
	vals := p.Values()
	i := slices.Index(vals, v)
	if i < 0 {
		return vals[0]
	}
	return vals[((i+step)%len(vals)+len(vals))%len(vals)]
}
