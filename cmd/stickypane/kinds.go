package main

import (
	"fmt"
	"io"
	"strings"

	"github.com/LeeSwallow/stickypane/internal/api"
	"github.com/LeeSwallow/stickypane/internal/initcmd"
	"github.com/LeeSwallow/stickypane/internal/kinds"
	"github.com/LeeSwallow/stickypane/internal/widget"
	"github.com/LeeSwallow/stickypane/internal/widget/note"
)

// folderShape is a shape that is a folder rather than a kind of file.
type folderShape struct{ name, blurb, command, usage, example string }

var folderShapes = []folderShape{
	{
		name:    "tab",
		blurb:   "A folder of the notes folder: a screen of its own, switched with 1-9.",
		command: "stickypane write deploy/notes --open",
		usage:   "The files in the folder are the tab's notes. Its sticky.json can give it a title and keeps how its notes are arranged. `stickypane show deploy/run.sh` switches to it.",
		example: ".sticky/deploy/\n  sticky.json   {\"title\": \"Deploy\"}\n  notes.md\n  run.sh",
	},
	{
		name:    "book",
		blurb:   "A folder inside a tab: one note whose pages are its files.",
		command: "stickypane link docs",
		usage:   "It scrolls like one long note, each page under a rule with its name; \",\" and \".\" turn the pages. `stickypane link <folder>` shows a folder of the project as a book without copying it.",
		example: ".sticky/deploy/guide/\n  01-intro.md\n  02-usage.md",
	},
}

// kindsCmd prints the shapes a note can take: all of them in short, or one
// in full. It needs no board, so an agent can read it before there is one.
func kindsCmd(e env, args []string) error {
	stdout := e.stdout
	reg := kinds.Default(note.Plain)
	if len(args) == 0 {
		for _, k := range reg {
			fmt.Fprintf(stdout, "%-10s %s %s\n%-10s   %s\n", k.Name, k.Icon, k.Blurb, "", k.Command)
		}
		for _, f := range folderShapes {
			fmt.Fprintf(stdout, "%-10s ▤ %s\n%-10s   %s\n", f.name, f.blurb, "", f.command)
		}
		_, err := fmt.Fprintln(stdout, "\nRun `stickypane kinds <shape>` for one in full.")
		return err
	}
	if len(args) > 1 {
		return usageError{cmd: "kinds"}
	}
	want := strings.ToLower(strings.TrimSpace(args[0]))
	for _, k := range reg {
		if k.Name == want {
			printShape(stdout, k.Name, k.Blurb, k.Command, k.Usage, k.ExampleFile(), keysOf(k))
			return nil
		}
	}
	for _, f := range folderShapes {
		if f.name == want {
			printShape(stdout, f.name, f.blurb, f.command, f.usage, f.example, "")
			return nil
		}
	}
	names := make([]string, 0, len(reg)+len(folderShapes))
	for _, k := range reg {
		names = append(names, k.Name)
	}
	for _, f := range folderShapes {
		names = append(names, f.name)
	}
	return usagef("kinds", "no shape is called %q; there are %s", want, strings.Join(names, ", "))
}

// keysOf spells out a kind's keys: "h l j k move · H L shift card".
func keysOf(k widget.Kind) string {
	var parts []string
	for i := 0; i+1 < len(k.Hint); i += 2 {
		parts = append(parts, k.Hint[i]+" "+k.Hint[i+1])
	}
	return strings.Join(parts, " · ")
}

func printShape(w io.Writer, name, blurb, command, usage, example, keys string) {
	fmt.Fprintf(w, "%s: %s\n\nMake one:\n  %s\n\nThe file:\n", name, blurb, command)
	for _, l := range strings.Split(strings.TrimRight(example, "\n"), "\n") {
		fmt.Fprintln(w, "  "+l)
	}
	fmt.Fprintf(w, "\nHow it behaves:\n  %s\n", usage)
	if keys != "" {
		fmt.Fprintf(w, "\nKeys on the board:\n  %s\n", keys)
	}
}

// guideCmd prints the agent guide in layers: the overview with the skills
// and shapes it names, or one of them in full. It needs no board.
func guideCmd(e env, args []string) error {
	if len(args) > 1 {
		return usageError{cmd: "guide"}
	}
	topic := ""
	if len(args) == 1 {
		topic = args[0]
	}
	text, err := api.New(nil, kinds.Default(note.Plain)).Guide(topic, initcmd.GuideText())
	if err != nil {
		return usagef("guide", "%v", err)
	}
	_, err = io.WriteString(e.stdout, strings.TrimRight(text, "\n")+"\n")
	return err
}
