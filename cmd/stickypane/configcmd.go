package main

import (
	"fmt"
	"text/tabwriter"

	"github.com/LeeSwallow/stickypane/internal/api"
	"github.com/LeeSwallow/stickypane/internal/prefs"
)

// configCmd lists the settings, prints one, or sets one: the settings
// panel (S on the board) for scripts and agents.
func configCmd(e env, args []string) error {
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
		w := tabwriter.NewWriter(e.stdout, 0, 0, 2, ' ', 0)
		for _, p := range prefs.All {
			v := prefs.Get(set, p.Key)
			mark := ""
			if v == p.Default {
				mark = "(default)"
			}
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", p.Key, v, mark, p.Help)
		}
		return w.Flush()
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
