package main

import (
	"fmt"
	"strings"
	"text/tabwriter"

	environment "github.com/LeeSwallow/stickypane/internal/env"
	"github.com/LeeSwallow/stickypane/internal/i18n"
)

// envReport is what `stickypane env` found, as --json prints it.
type envReport struct {
	System     string            `json:"system"`
	Shell      string            `json:"shell"`
	Pane       string            `json:"pane"`
	Split      string            `json:"split"`
	Language   string            `json:"language"`
	LocaleFrom string            `json:"locale_from"`
	Locale     string            `json:"locale"`
	Scripts    map[string]string `json:"scripts"`
	Editor     []string          `json:"editor"`
}

// envCmd says where the board runs and what follows from it. It needs no
// board, so it is the first thing to run when something looks wrong.
func envCmd(e env, args []string) error {
	fs := newFlags("env")
	asJSON := fs.Bool("json", false, "print what was found as JSON")
	if _, err := parse(fs, "env", args, 0, 0); err != nil {
		return err
	}
	d := environment.Detect()
	r := envReport{
		System:     d.OS,
		Shell:      d.Shell.Name,
		Pane:       d.Pane.Name,
		Split:      d.Pane.Split("stickypane"),
		Language:   i18n.Detect(),
		LocaleFrom: d.LocaleFrom,
		Locale:     d.Locale,
		Scripts:    map[string]string{},
		Editor:     d.Editor(),
	}
	for _, ext := range []string{".sh", ".ps1"} {
		prog, _, err := d.Script("script" + ext)
		if err != nil {
			r.Scripts[ext] = "not available: " + err.Error()
		} else {
			r.Scripts[ext] = prog
		}
	}
	if *asJSON {
		return printJSON(e.stdout, r)
	}
	w := tabwriter.NewWriter(e.stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintf(w, "system\t%s\n", r.System)
	fmt.Fprintf(w, "shell\t%s\n", r.Shell)
	if r.Split != "" {
		fmt.Fprintf(w, "pane\t%s: `%s` opens the board next to your agent\n", r.Pane, r.Split)
	} else {
		fmt.Fprintf(w, "pane\tno tmux, Zellij, WezTerm or Windows Terminal here: run `stickypane` in a terminal beside your agent\n")
	}
	from := "nothing set, so English"
	if r.LocaleFrom != "" {
		from = fmt.Sprintf("from %s=%s", r.LocaleFrom, r.Locale)
		if r.LocaleFrom == "system" {
			from = "from the system's locale " + r.Locale
		}
	}
	other := "ko"
	if r.Language == "ko" {
		other = "en"
	}
	fmt.Fprintf(w, "language\t%s (%s); to change it: `%s`, or `stickypane language %s`\n", r.Language, from, d.Shell.SetVar("STICKYPANE_LANG", other), other)
	fmt.Fprintf(w, "scripts\t.sh: %s\n", r.Scripts[".sh"])
	fmt.Fprintf(w, "\t.ps1: %s\n", r.Scripts[".ps1"])
	fmt.Fprintf(w, "editor\t%s\n", strings.Join(r.Editor, " "))
	return w.Flush()
}
