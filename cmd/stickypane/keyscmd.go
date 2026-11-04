package main

import (
	"fmt"
	"maps"
	"slices"
	"strings"
	"text/tabwriter"

	"github.com/LeeSwallow/stickypane/internal/adapter/keyfile"
	"github.com/LeeSwallow/stickypane/internal/surface/app"
)

// keysCmd lists the board's keys, or changes one action's keys in
// keys.json, or takes the changes back. It needs no board.
func keysCmd(e env, args []string) error {
	fs := newFlags("keys")
	asJSON := fs.Bool("json", false, "print the keys as JSON")
	words, err := parse(fs, "keys", args, 0, -1)
	if err != nil {
		return err
	}
	path := keyfile.Path()
	shown := keyfile.Shown(path)
	switch {
	case len(words) == 0:
		return listKeys(e, path, shown, *asJSON)
	case words[0] == "set" && len(words) >= 2:
		return setKeys(e, path, shown, words[1], words[2:])
	case words[0] == "reset" && len(words) <= 2:
		return resetKeys(e, path, shown, words[1:])
	}
	return usageError{cmd: "keys"}
}

// listKeys prints every action: its name, its keys, what it does, and
// whether keys.json changed it. What of the file was not used goes to
// standard error.
func listKeys(e env, path, shown string, asJSON bool) error {
	want, err := keyfile.Read(path)
	if err != nil {
		fmt.Fprintf(e.stderr, "%s: %v; the keys are as they come\n", shown, err)
	}
	bindings, problems := app.Bindings(helpStrings(), want)
	for _, p := range problems {
		fmt.Fprintf(e.stderr, "%s: %s; it keeps the keys it comes with\n", shown, p)
	}
	if asJSON {
		return printJSON(e.stdout, bindings)
	}
	// The words come last, where a wide character cannot push a column
	// out of line; the mark comes before them, when there is one at all.
	marked := slices.ContainsFunc(bindings, func(b app.Binding) bool { return b.Changed })
	w := tabwriter.NewWriter(e.stdout, 0, 0, 2, ' ', 0)
	for _, b := range bindings {
		fmt.Fprintf(w, "%s\t%s\t", b.Name, shownKeys(b.Keys))
		if marked {
			mark := ""
			if b.Changed {
				mark = "(" + keyfile.Name + ")"
			}
			fmt.Fprintf(w, "%s\t", mark)
		}
		fmt.Fprintln(w, b.Words)
	}
	return w.Flush()
}

// setKeys gives an action keys in keys.json, keeping the rest of the
// file; no keys at all leaves it without one. Nothing is written when the
// board could not use them.
func setKeys(e env, path, shown, name string, keys []string) error {
	want, err := keyfile.Read(path)
	if err != nil {
		return fmt.Errorf("%s: %v; stickypane keys reset starts it again", shown, err)
	}
	normal, err := app.NormalKeys(keys)
	if err != nil {
		return fmt.Errorf("%s: %v", name, err)
	}
	next := maps.Clone(want)
	if next == nil {
		next = map[string][]string{}
	}
	next[name] = normal
	bindings, problems := app.Bindings(helpStrings(), next)
	for _, p := range problems {
		if p.Action == name {
			return fmt.Errorf("%s; %s is left as it was", p, shown)
		}
	}
	if err := keyfile.Write(path, next); err != nil {
		return err
	}
	return printKeysOf(e, bindings, name)
}

// resetKeys takes an action's change out of keys.json, or, with no
// action, removes the file, so the keys are as they come.
func resetKeys(e env, path, shown string, names []string) error {
	if len(names) == 0 {
		if err := keyfile.Remove(path); err != nil {
			return err
		}
		_, err := fmt.Fprintf(e.stdout, "%s removed; the keys are as they come\n", shown)
		return err
	}
	name := names[0]
	want, err := keyfile.Read(path)
	if err != nil {
		return fmt.Errorf("%s: %v; stickypane keys reset removes it", shown, err)
	}
	bindings, _ := app.Bindings(helpStrings(), nil)
	if _, ok := want[name]; ok {
		delete(want, name)
		if len(want) == 0 {
			err = keyfile.Remove(path)
		} else {
			err = keyfile.Write(path, want)
		}
		if err != nil {
			return err
		}
		bindings, _ = app.Bindings(helpStrings(), want)
	}
	return printKeysOf(e, bindings, name)
}

// printKeysOf prints the keys the action called name has now.
func printKeysOf(e env, bindings []app.Binding, name string) error {
	for _, b := range bindings {
		if b.Name == name {
			_, err := fmt.Fprintf(e.stdout, "%s: %s\n", name, shownKeys(b.Keys))
			return err
		}
	}
	return fmt.Errorf("there is no action %q; stickypane keys lists them", name)
}

// shownKeys is keys as the board shows them, side by side: the sequence
// "g g" is gg.
func shownKeys(keys []string) string {
	out := make([]string, len(keys))
	for i, k := range keys {
		out[i] = app.ShowKey(k)
	}
	return strings.Join(out, " ")
}
