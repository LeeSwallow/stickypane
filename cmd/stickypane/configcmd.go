package main

import (
	"fmt"
	"io"
	"strings"

	"github.com/charmbracelet/x/ansi"

	"github.com/LeeSwallow/stickypane/internal/core/i18n"
	"github.com/LeeSwallow/stickypane/internal/usecase/api"
	"github.com/LeeSwallow/stickypane/internal/usecase/prefs"
)

// configCmd lists the settings, prints one, or sets one: the settings
// panel (S on the board) for scripts and agents. Called as theme or
// language, it is that setting's choice (settings.go).
func configCmd(e env, args []string) error {
	switch e.cmd {
	case "theme":
		return themeCmd(e, args)
	case "language":
		return languageCmd(e, args)
	}
	words, err := parse(newFlags("config"), "config", args, 0, 2)
	if err != nil {
		return err
	}
	a, err := open(e, false)
	if err != nil {
		return err
	}
	set := a.Settings()
	switch len(words) {
	case 0:
		return listSettings(e.stdout, helpStrings(), func(key string) string { return prefs.Get(set, key) })
	case 1:
		if prefs.Lookup(words[0]).Key == "" {
			_, err := prefs.Normal(words[0], "")
			return err // there is no such setting
		}
		_, err := fmt.Fprintln(e.stdout, prefs.Get(set, words[0]))
		return err
	}
	if err := a.ChangeSettings(api.Setting{Key: words[0], Value: words[1]}); err != nil {
		return err
	}
	_, err = fmt.Fprintf(e.stdout, "%s: %s\n", words[0], prefs.Get(a.Settings(), words[0]))
	return err
}

// listSettings prints every setting with its value, whether that is the
// default, and what it changes: the panel's first, then the rest under
// "more:", in one set of columns. Keys and values are as they are typed;
// the rest is in tr's language.
func listSettings(w io.Writer, tr i18n.Strings, get func(key string) string) error {
	def := tr.Say("config.default", nil)
	keyW, valW, markW := 0, 0, ansi.StringWidth(def)
	for _, p := range prefs.All {
		keyW, valW = max(keyW, len(p.Key)), max(valW, len(get(p.Key)))
	}
	var first, more strings.Builder
	for _, p := range prefs.All {
		b := &first
		if p.More {
			b = &more
		}
		v, mark := get(p.Key), ""
		if v == p.Default {
			mark = def
		}
		mark += strings.Repeat(" ", markW-ansi.StringWidth(mark))
		fmt.Fprintf(b, "%-*s  %-*s  %s  %s\n", keyW, p.Key, valW, v, mark, tr.L(p.Help))
	}
	_, err := io.WriteString(w, first.String()+"\n"+tr.Say("config.more", nil)+"\n"+more.String())
	return err
}
