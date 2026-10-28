package main

import (
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/LeeSwallow/stickypane/internal/api"
	"github.com/LeeSwallow/stickypane/internal/i18n"
	"github.com/LeeSwallow/stickypane/internal/prefs"
	"github.com/LeeSwallow/stickypane/internal/theme"
)

// choice is a setting of the root sticky.json the user picks from a list:
// the theme or the language. Which values it takes is the settings' rule
// (prefs.Normal), and it is written as every front end writes settings
// (api.ChangeSettings); the command only says so.
type choice struct {
	cmd     string                                      // the command, which is the setting's key
	list    func(w io.Writer, mark func(string) string) // every option, marked
	missing string                                      // "there is no theme %q. The themes are:"
}

// run lists the options with the current one marked, or chooses one.
func (c choice) run(e env, args []string) error {
	words, err := parse(newFlags(c.cmd), c.cmd, args, 0, 1)
	if err != nil {
		return err
	}
	a, err := open(e, false)
	if err != nil {
		return err
	}
	current := a.Settings().Root(c.cmd)
	if current == "" {
		current = "auto"
	}
	list := func(w io.Writer) {
		c.list(w, func(name string) string {
			if strings.EqualFold(name, current) {
				return "* "
			}
			return "  "
		})
	}
	if len(words) == 0 {
		list(e.stdout)
		return nil
	}
	name, err := prefs.Normal(c.cmd, words[0])
	var refused *prefs.Refused
	if errors.As(err, &refused) {
		var b strings.Builder
		fmt.Fprintf(&b, c.missing+"\n", words[0])
		list(&b)
		return errors.New(strings.TrimRight(b.String(), "\n"))
	} else if err != nil {
		return err
	}
	if err := a.ChangeSettings(api.Setting{Key: c.cmd, Value: name}); err != nil {
		return err
	}
	_, err = fmt.Fprintf(e.stdout, "%s: %s\n", c.cmd, name)
	return err
}

// themeCmd lists the themes or chooses one.
func themeCmd(e env, args []string) error {
	return choice{
		cmd: "theme",
		list: func(w io.Writer, mark func(string) string) {
			fmt.Fprintf(w, "%sauto  (follows the terminal: %s or %s)\n", mark("auto"), theme.DefaultDark, theme.DefaultLight)
			for _, t := range theme.All() {
				kind := "dark"
				if !t.Dark {
					kind = "light"
				}
				fmt.Fprintf(w, "%s%s  (%s)\n", mark(t.Name), t.Name, kind)
			}
		},
		missing: "there is no theme %q. The themes are:",
	}.run(e, args)
}

// languageCmd lists the languages the screen can speak or chooses one.
func languageCmd(e env, args []string) error {
	return choice{
		cmd: "language",
		list: func(w io.Writer, mark func(string) string) {
			fmt.Fprintf(w, "%sauto  (follows STICKYPANE_LANG, LC_ALL, LC_MESSAGES, LANG; now %s)\n", mark("auto"), i18n.Detect())
			for _, code := range i18n.Languages() {
				fmt.Fprintf(w, "%s%s\n", mark(code), code)
			}
		},
		missing: "there is no translation for %q. The languages are:",
	}.run(e, args)
}
