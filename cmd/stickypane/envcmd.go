package main

import (
	"fmt"
	"strings"
	"text/tabwriter"

	environment "github.com/LeeSwallow/stickypane/internal/adapter/env"
	"github.com/LeeSwallow/stickypane/internal/core/i18n"
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
		Language:   i18n.Detect(d.Locale),
		LocaleFrom: d.LocaleFrom,
		Locale:     d.Locale,
		Scripts:    map[string]string{},
		Editor:     d.Editor(),
	}
	// The rows are named as --json names them; what follows each is in
	// the person's language, and --json stays English.
	tr := helpStrings()
	shown := map[string]string{}
	for _, ext := range []string{".sh", ".ps1"} {
		prog, _, err := d.Script("script" + ext)
		if err != nil {
			r.Scripts[ext] = "not available: " + err.Error()
			shown[ext] = tr.Say("env.noScript", map[string]any{"Err": err})
		} else {
			r.Scripts[ext], shown[ext] = prog, prog
		}
	}
	if *asJSON {
		return printJSON(e.stdout, r)
	}
	w := tabwriter.NewWriter(e.stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintf(w, "system\t%s\n", r.System)
	fmt.Fprintf(w, "shell\t%s\n", r.Shell)
	if r.Split != "" {
		fmt.Fprintf(w, "pane\t%s\n", tr.Say("env.split", map[string]any{"Pane": r.Pane, "Split": r.Split}))
	} else {
		fmt.Fprintf(w, "pane\t%s\n", tr.Say("env.noPane", nil))
	}
	from := tr.Say("env.langNone", nil)
	if r.LocaleFrom != "" {
		from = tr.Say("env.langFrom", map[string]any{"Var": r.LocaleFrom, "Value": r.Locale})
		if r.LocaleFrom == environment.FromSystem {
			from = tr.Say("env.langSystem", map[string]any{"Locale": r.Locale})
		}
	}
	other := "ko"
	if r.Language == "ko" {
		other = "en"
	}
	fmt.Fprintf(w, "language\t%s\n", tr.Say("env.language", map[string]any{"Language": r.Language, "From": from, "Set": d.Shell.SetVar(environment.LangVar, other), "Other": other}))
	fmt.Fprintf(w, "scripts\t.sh: %s\n", shown[".sh"])
	fmt.Fprintf(w, "\t.ps1: %s\n", shown[".ps1"])
	fmt.Fprintf(w, "editor\t%s\n", strings.Join(r.Editor, " "))
	return w.Flush()
}
