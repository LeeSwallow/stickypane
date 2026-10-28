package main

import (
	"fmt"
	"io"
	"strings"

	"github.com/LeeSwallow/stickypane/internal/i18n"
	"github.com/LeeSwallow/stickypane/internal/kinds"
	"github.com/LeeSwallow/stickypane/internal/store"
	"github.com/LeeSwallow/stickypane/internal/widget"
	"github.com/LeeSwallow/stickypane/internal/widget/note"
)

// helpText is what `stickypane help <command>` says about one command. It
// sits in the command table beside the command, in English; a language
// overlays it in internal/i18n under the keys "<name>.brief", "<name>.what"
// and "<name>.args". Examples are command lines, never translated, and a
// test runs each one.
type helpText struct {
	group    string   // where the overview lists it: one of helpGroups
	brief    string   // a line for the overview
	what     string   // what it does, a sentence or two
	syntax   string   // the usage line, without the prose
	args     string   // arguments and options, a line each; may be empty
	examples []string // two or three command lines that work
	related  []string // other commands, shapes or "start"
}

// helpGroups are the tasks the overview groups the commands by.
var helpGroups = []struct{ key, title string }{
	{"start", "Start"},
	{"notes", "Notes"},
	{"arrange", "Arrange"},
	{"settings", "Settings"},
	{"agents", "Agents"},
	{"scripts", "Scripts"},
}

// startText is the first steps, in English.
const startText = `The way to start

1. In your project, make the board and tell your agents about it:
     stickypane init
2. Put a note on it. A script or an agent writes the file; so can you:
     echo "- [ ] try the board" | stickypane write plan --type checklist --open
   (stickypane kinds lists the shapes a note can take)
3. Open the board in a pane beside your agent:
     stickypane
   (stickypane env says the command for your terminal)
4. Connect an agent, once:
     stickypane setup
   (or stickypane setup --mcp, or the lines init put in CLAUDE.md / AGENTS.md)
5. The desktop app shows the same board on your screen. It starts
   stickypane bridge by itself; there is nothing to type.

Next: stickypane help, then stickypane help <command>.`

const overviewHead = "stickypane - a note board for you and your coding agent"

const overviewFoot = `stickypane help <command>   one command, with examples
stickypane help <shape>     a note's file format and an example
stickypane help start       the way to start
stickypane help --all       every command with every option`

// helpStrings are the words of the help: the board's language when there
// is a board and it chose one, else what the environment asks for.
func helpStrings() i18n.Strings {
	choice := ""
	if dir, err := store.Resolve("."); err == nil {
		choice = store.Open(dir).Language()
	}
	return i18n.Pick(choice)
}

// hasBoard says whether there is a board here or above.
func hasBoard() bool {
	_, err := store.Resolve(".")
	return err == nil
}

// helpCmd prints the overview, the help of one command or shape, or the
// way to start. It needs no board.
func helpCmd(e env, args []string) error {
	fs := newFlags("help")
	all := fs.Bool("all", false, "print every command with every option")
	words, err := parse(fs, "help", args, 0, 1)
	if err != nil {
		return err
	}
	s := helpStrings()
	switch {
	case *all:
		_, err = io.WriteString(e.stdout, usage)
	case len(words) == 0:
		if !hasBoard() {
			fmt.Fprintln(e.stdout, s.T("start", startText))
			fmt.Fprintln(e.stdout)
		}
		_, err = io.WriteString(e.stdout, overview(s))
	default:
		err = helpOn(e.stdout, s, strings.ToLower(strings.TrimSpace(words[0])))
	}
	return err
}

// helpOn prints the help of one thing a person can ask about.
func helpOn(w io.Writer, s i18n.Strings, want string) error {
	if want == "start" {
		_, err := fmt.Fprintln(w, s.T("start", startText))
		return err
	}
	if c, ok := lookup(want); ok {
		_, err := io.WriteString(w, commandHelp(s, c))
		return err
	}
	if k, f, ok := findShape(want); ok {
		if k != nil {
			printShape(w, s, k.Name, k.Blurb, k.Command, k.Usage, k.ExampleFile(), keysOf(*k))
		} else {
			printShape(w, s, f.name, f.blurb, f.command, f.usage, f.example, "")
		}
		return nil
	}
	msg := fmt.Sprintf("no command or shape is called %q", want)
	if near := closest(want); near != "" {
		msg += fmt.Sprintf("; did you mean %q? (stickypane help %s)", near, near)
	} else {
		msg += "; stickypane help lists them"
	}
	return usagef("help", "%s", msg)
}

// findShape finds the shape a word names: a kind of note or a folder.
func findShape(want string) (*widget.Kind, *folderShape, bool) {
	reg := kinds.Default(note.Plain)
	for i := range reg {
		if reg[i].Name == want {
			return &reg[i], nil, true
		}
	}
	for i := range folderShapes {
		if folderShapes[i].name == want {
			return nil, &folderShapes[i], true
		}
	}
	return nil, nil, false
}

// names are everything help can be asked about.
func helpNames() []string {
	var out []string
	for _, c := range commands {
		out = append(out, c.names[0])
	}
	for _, k := range kinds.Default(note.Plain) {
		out = append(out, k.Name)
	}
	for _, f := range folderShapes {
		out = append(out, f.name)
	}
	return append(out, "start")
}

// closest returns the name nearest to want, or "" when none is near.
func closest(want string) string {
	best, bestDist := "", 1<<30
	for _, n := range helpNames() {
		d := distance(want, n)
		if len(want) >= 2 && strings.HasPrefix(n, want) {
			d = 1
		}
		if d < bestDist {
			best, bestDist = n, d
		}
	}
	if bestDist <= 2 || bestDist*3 <= len(want) {
		return best
	}
	return ""
}

// distance is the edit distance between two words.
func distance(a, b string) int {
	ra, rb := []rune(a), []rune(b)
	prev := make([]int, len(rb)+1)
	for j := range prev {
		prev[j] = j
	}
	for i := 1; i <= len(ra); i++ {
		cur := make([]int, len(rb)+1)
		cur[0] = i
		for j := 1; j <= len(rb); j++ {
			cost := 1
			if ra[i-1] == rb[j-1] {
				cost = 0
			}
			cur[j] = min(prev[j]+1, cur[j-1]+1, prev[j-1]+cost)
		}
		prev = cur
	}
	return prev[len(rb)]
}

// commandHelp is the help of one command.
func commandHelp(s i18n.Strings, c command) string {
	name, h := c.names[0], c.help
	var b strings.Builder
	fmt.Fprintf(&b, "stickypane %s - %s\n\n%s\n\n", name, s.T(name+".brief", h.brief), s.T(name+".what", h.what))
	fmt.Fprintf(&b, "%s:\n", s.T("label.usage", "Usage"))
	b.WriteString("  " + h.syntax + "\n")
	if h.args != "" {
		fmt.Fprintf(&b, "\n%s:\n", s.T("label.options", "Arguments and options"))
		for _, l := range strings.Split(s.T(name+".args", h.args), "\n") {
			b.WriteString("  " + l + "\n")
		}
	}
	fmt.Fprintf(&b, "\n%s:\n", s.T("label.examples", "Examples"))
	for _, ex := range h.examples {
		b.WriteString("  " + ex + "\n")
	}
	if len(h.related) > 0 {
		fmt.Fprintf(&b, "\n%s: %s  (stickypane help <%s>)\n", s.T("label.related", "Related"), strings.Join(h.related, ", "), s.T("label.name", "name"))
	}
	return b.String()
}

// overview is the short help, grouped by task.
func overview(s i18n.Strings) string {
	var b strings.Builder
	b.WriteString(s.T("overview.head", overviewHead) + "\n")
	for _, g := range helpGroups {
		fmt.Fprintf(&b, "\n%s\n", s.T("group."+g.key, g.title))
		if g.key == "start" {
			fmt.Fprintf(&b, "  %-9s %s\n", "[path]", s.T("open.brief", "open the board of the project at path (default: here)"))
		}
		for _, c := range commands {
			if c.help.group == g.key {
				name := c.names[0]
				fmt.Fprintf(&b, "  %-9s %s\n", name, s.T(name+".brief", c.help.brief))
			}
		}
	}
	b.WriteString("\n" + s.T("overview.foot", overviewFoot) + "\n")
	return b.String()
}
