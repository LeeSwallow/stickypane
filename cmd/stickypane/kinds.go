package main

import (
	"fmt"
	"io"
	"strings"

	"github.com/LeeSwallow/stickypane/internal/core/i18n"
	"github.com/LeeSwallow/stickypane/internal/core/kinds"
	"github.com/LeeSwallow/stickypane/internal/core/widget"
	"github.com/LeeSwallow/stickypane/internal/core/widget/note"
	"github.com/LeeSwallow/stickypane/internal/usecase/api"
	"github.com/LeeSwallow/stickypane/internal/usecase/initcmd"
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
	if k, f, ok := findShape(want); ok {
		if k != nil {
			printShape(stdout, i18n.English(), k.Name, k.Blurb, k.Command, k.Usage, k.ExampleFile(), keysOf(*k), optionsOf(*k))
		} else {
			printShape(stdout, i18n.English(), f.name, f.blurb, f.command, f.usage, f.example, "", "")
		}
		return nil
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

// optionsOf spells out a kind's create options, one a line, as `write
// --opt` takes them: "--opt view=bar|spark|heat  default: bar".
func optionsOf(k widget.Kind) string {
	var lines []string
	for _, o := range k.Options {
		value := "<" + o.Type + ">"
		if o.Type == widget.OptChoice {
			value = strings.Join(o.Choices, "|")
		}
		line := "--opt " + o.Name + "=" + value
		if o.Default != "" {
			line += "  default: " + strings.ReplaceAll(o.Default, "\n", " | ")
		}
		lines = append(lines, line)
	}
	return strings.Join(lines, "\n  ")
}

// printShape prints a shape in full; the same text answers `kinds <shape>`
// and `help <shape>`, the headings in the language of s.
func printShape(w io.Writer, s i18n.Strings, name, blurb, command, usage, example, keys, options string) {
	fmt.Fprintf(w, "%s: %s\n\n%s:\n  %s\n\n%s:\n", name, blurb, s.T("shape.make", "Make one"), command, s.T("shape.file", "The file"))
	for _, l := range strings.Split(strings.TrimRight(example, "\n"), "\n") {
		fmt.Fprintln(w, "  "+l)
	}
	fmt.Fprintf(w, "\n%s:\n  %s\n", s.T("shape.behaves", "How it behaves"), usage)
	if keys != "" {
		fmt.Fprintf(w, "\n%s:\n  %s\n", s.T("shape.keys", "Keys on the board"), keys)
	}
	if options != "" {
		fmt.Fprintf(w, "\n%s:\n  %s\n", s.T("shape.options", "Make one from options (write --type, the desktop app)"), options)
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
