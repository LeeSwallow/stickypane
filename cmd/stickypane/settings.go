package main

import (
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/LeeSwallow/stickypane/internal/i18n"
	"github.com/LeeSwallow/stickypane/internal/store"
	"github.com/LeeSwallow/stickypane/internal/theme"
)

// choice is a setting of the root sticky.json the user picks from a list:
// the theme or the language.
type choice struct {
	cmd     string
	current func(*store.Store) string
	set     func(*store.Store, string) error
	list    func(w io.Writer, mark func(string) string) // every option, marked
	resolve func(arg string) (string, bool)             // the option arg names
	missing string                                      // "there is no theme %q. The themes are:"
}

// run lists the options with the current one marked, or chooses one.
func (c choice) run(e env, args []string) error {
	words, err := parse(newFlags(c.cmd), c.cmd, args, 0, 1)
	if err != nil {
		return err
	}
	dir, err := store.Resolve(".")
	if errors.Is(err, store.ErrNotFound) {
		return errNoBoard
	}
	if err != nil {
		return err
	}
	st := store.Open(dir)
	current := c.current(st)
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
	name, ok := c.resolve(strings.TrimSpace(words[0]))
	if !ok {
		var b strings.Builder
		fmt.Fprintf(&b, c.missing+"\n", words[0])
		list(&b)
		return errors.New(strings.TrimRight(b.String(), "\n"))
	}
	if err := c.set(st, name); err != nil {
		return err
	}
	_, err = fmt.Fprintf(e.stdout, "%s: %s\n", c.cmd, name)
	return err
}

// themeCmd lists the themes or chooses one.
func themeCmd(e env, args []string) error {
	return choice{
		cmd:     "theme",
		current: (*store.Store).Theme,
		set:     (*store.Store).SetTheme,
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
		resolve: func(arg string) (string, bool) {
			if strings.EqualFold(arg, "auto") {
				return "auto", true
			}
			t, ok := theme.Lookup(arg)
			return t.Name, ok
		},
		missing: "there is no theme %q. The themes are:",
	}.run(e, args)
}

// languageCmd lists the languages the screen can speak or chooses one.
func languageCmd(e env, args []string) error {
	return choice{
		cmd:     "language",
		current: (*store.Store).Language,
		set:     (*store.Store).SetLanguage,
		list: func(w io.Writer, mark func(string) string) {
			fmt.Fprintf(w, "%sauto  (follows STICKYPANE_LANG, LC_ALL, LC_MESSAGES, LANG; now %s)\n", mark("auto"), i18n.Detect())
			for _, code := range i18n.Languages() {
				fmt.Fprintf(w, "%s%s\n", mark(code), code)
			}
		},
		resolve: func(arg string) (string, bool) {
			code := strings.ToLower(arg)
			if code == "auto" {
				return code, true
			}
			_, err := i18n.Load(code)
			return code, err == nil
		},
		missing: "there is no translation for %q. The languages are:",
	}.run(e, args)
}
