package main

import (
	"context"
	"fmt"
	"io"
	"path/filepath"
	"time"

	"github.com/LeeSwallow/stickypane/internal/initcmd"
	"github.com/LeeSwallow/stickypane/internal/registry"
	"github.com/LeeSwallow/stickypane/internal/store"
)

// env is what every command gets: the context, which ends when the user
// interrupts, and the standard streams. Tests hand in buffers.
type env struct {
	ctx    context.Context
	stdin  io.Reader
	stdout io.Writer
	stderr io.Writer
	cmd    string           // the word of the command line that chose the command
	now    func() time.Time // the clock every time written is read from
}

// command is one word of the command line. Its usage is the lines of the
// usage text that start with "stickypane <name>"; its help sits beside it
// here, so the table and the help cannot drift apart (a test holds them).
type command struct {
	names []string // the name and its aliases
	run   func(e env, args []string) error
	help  helpText
}

// commands is the table run looks words up in, in the order of the usage
// text. It is filled in init, because help reads it.
var commands []command

func init() {
	commands = []command{
		{[]string{"init"}, initCmd, helpText{
			group: "start", brief: "make .sticky/ and tell your agents about it",
			what:     "Makes .sticky/ in this folder and puts the agent guide where agents read it: CLAUDE.md, AGENTS.md or a .claude skill. Safe to run again.",
			syntax:   "stickypane init [--skill] [--no-agent-docs]",
			args:     "--skill          a Claude Code skill instead of the instruction files\n--no-agent-docs  the folder only",
			examples: []string{"stickypane init", "stickypane init --skill", "stickypane init --no-agent-docs"},
			related:  []string{"setup", "guide", "start"},
		}},
		{[]string{"setup"}, setupCmd, helpText{
			group: "agents", brief: "install the board plugin for Claude Code and Codex",
			what:     "Installs the board plugin through each agent's own commands, and asks first when run in a terminal. It makes no board: run stickypane init in a project, or let the agent's first change make one in a git repository.",
			syntax:   "stickypane setup [--scope user|project|local] [--mcp] [--agents claude,codex] [--undo] [--dry-run] [--yes]",
			args:     "--scope S    user, project or local (Claude Code)\n--mcp        also register stickypane as an MCP server\n--agents L   claude, codex or both (default: those found)\n--undo       uninstall the plugin (and the MCP server with --mcp)\n--dry-run    print the plan and run nothing\n--yes        run without asking",
			examples: []string{"stickypane setup --dry-run", "stickypane setup --mcp --yes", "stickypane setup --undo"},
			related:  []string{"init", "mcp", "guide"},
		}},
		{[]string{"theme"}, themeCmd, helpText{
			group: "settings", brief: "list the themes or choose one",
			what:     "Lists the themes with the current one marked, or chooses one. \"auto\" follows the terminal's background.",
			syntax:   "stickypane theme [name]",
			examples: []string{"stickypane theme", "stickypane theme nord", "stickypane theme auto"},
			related:  []string{"config", "language"},
		}},
		{[]string{"language"}, languageCmd, helpText{
			group: "settings", brief: "list the screen's languages or choose one",
			what:     "Lists the languages of the screen, or chooses one. \"auto\" follows LANG. Help follows it too; the agent guide stays English.",
			syntax:   "stickypane language [code]",
			examples: []string{"stickypane language", "stickypane language ko", "stickypane language auto"},
			related:  []string{"config", "theme", "env"},
		}},
		{[]string{"config"}, configCmd, helpText{
			group: "settings", brief: "list, read or change a setting",
			what:     "Lists the settings with their values, prints one, or changes one: theme, language, time, editor, stale. Each has a default; S on the board shows them too.",
			syntax:   "stickypane config [key [value]]",
			examples: []string{"stickypane config", "stickypane config language", "stickypane config theme nord"},
			related:  []string{"theme", "language"},
		}},
		{[]string{"kinds"}, kindsCmd, helpText{
			group: "start", brief: "list the shapes a note can take",
			what:     "Lists the shapes a note can take, or shows one in full: the command that makes it, a file to copy and how it behaves. Needs no board.",
			syntax:   "stickypane kinds [shape]",
			examples: []string{"stickypane kinds", "stickypane kinds board", "stickypane kinds checklist"},
			related:  []string{"write", "guide", "checklist"},
		}},
		{[]string{"env"}, envCmd, helpText{
			group: "start", brief: "say what the board found about where it runs",
			what:     "Says what the board found: the system, the shell, how to open a pane beside the agent, the screen's language and how scripts run.",
			syntax:   "stickypane env [--json]",
			args:     "--json  print what was found as JSON",
			examples: []string{"stickypane env", "stickypane env --json"},
			related:  []string{"language", "start"},
		}},
		{[]string{"guide"}, guideCmd, helpText{
			group: "agents", brief: "print the agent guide",
			what:     "Prints the agent guide in layers: the overview, or one skill or shape in full. The guide is English. Needs no board.",
			syntax:   "stickypane guide [topic]",
			examples: []string{"stickypane guide", "stickypane guide tracking-progress", "stickypane guide checklist"},
			related:  []string{"setup", "mcp", "kinds"},
		}},
		{[]string{"version", "--version", "-v"}, func(e env, _ []string) error {
			_, err := fmt.Fprintln(e.stdout, "stickypane", fullVersion())
			return err
		}, helpText{
			group: "start", brief: "print the version",
			what:     "Prints the version of this program.",
			syntax:   "stickypane version",
			examples: []string{"stickypane version"},
			related:  []string{"env", "help"},
		}},
		{[]string{"help", "--help", "-h"}, helpCmd, helpText{
			group: "start", brief: "this help: a command, a shape, or getting started",
			what:     "Prints a short overview grouped by task, or the help of one command or shape, or the path to start with. An unknown name suggests the closest command.",
			syntax:   "stickypane help [command|shape|start] [--all]",
			args:     "--all  every command with every option, in one page",
			examples: []string{"stickypane help", "stickypane help card", "stickypane help checklist", "stickypane help start"},
			related:  []string{"kinds", "guide"},
		}},
		{[]string{"list"}, listCmd, helpText{
			group: "notes", brief: "list the notes",
			what:     "Lists the notes: name, title, shape, whether it is open, and its size.",
			syntax:   "stickypane list [--json]",
			args:     "--json  print the list as JSON",
			examples: []string{"stickypane list", "stickypane list --json"},
			related:  []string{"index", "cat", "show"},
		}},
		{[]string{"index"}, indexCmd, helpText{
			group: "notes", brief: "every note in a line: title, counts, a gist",
			what:     "Prints every note in a line, by tab: its title, what it counts and a gist. Read it before the notes.",
			syntax:   "stickypane index [--json]",
			args:     "--json  print the index as JSON",
			examples: []string{"stickypane index", "stickypane index --json"},
			related:  []string{"list", "cat"},
		}},
		{[]string{"show"}, manageCmd("show"), helpText{
			group: "notes", brief: "put a note, a file or a folder on the screen",
			what:     "Puts a note on the screen. A path to any file or folder of the project is linked onto the board without copying it.",
			syntax:   "stickypane show <name|path>",
			examples: []string{"stickypane show plan", "stickypane show README.md", "stickypane show docs/"},
			related:  []string{"hide", "link", "write"},
		}},
		{[]string{"hide"}, manageCmd("hide"), helpText{
			group: "notes", brief: "fold a note away",
			what:     "Folds a note away. The file stays; show brings it back.",
			syntax:   "stickypane hide <name>",
			examples: []string{"stickypane hide plan"},
			related:  []string{"show", "archive", "rm"},
		}},
		{[]string{"cat"}, catCmd, helpText{
			group: "notes", brief: "print a note's file",
			what:     "Prints a note's file as it is on disk.",
			syntax:   "stickypane cat <name>",
			examples: []string{"stickypane cat plan", "stickypane cat deploy/notes"},
			related:  []string{"list", "write"},
		}},
		{[]string{"write"}, writeCmd, helpText{
			group: "notes", brief: "create or replace a note from standard input",
			what:     "Creates or replaces a note with what comes on standard input, exactly as overwriting the file would. Read a note before rewriting it.",
			syntax:   "stickypane write <name> [--type T] [--title X] [--open] [--size S] [--opt k=v]",
			args:     "--type T   the shape: note, board, checklist, log, chart, form, script\n--title X  the note's title\n--open     put the note on the screen now\n--size S   the width: 1/4, 1/3, 1/2, 2/3, 3/4 or 1 of the screen (or card, half, page)\n--opt k=v  a create option of the shape (stickypane kinds lists them), repeated for more lines; the shape writes the note and standard input is not read",
			examples: []string{"echo \"- [ ] build\" | stickypane write plan --type checklist --title \"Release\" --open", "stickypane write sprint --type board --opt columns=Backlog --opt columns=Done", "echo \"hello\" | stickypane write hello", "echo \"- [ ] a\" | stickypane write deploy/steps --type checklist"},
			related:  []string{"kinds", "set", "cat"},
		}},
		{[]string{"todo"}, editCmd("todo"), helpText{
			group: "notes", brief: "add, tick or untick a checklist item",
			what:     "Changes one item of a checklist without reading or rewriting it, and prints where the note stands. An item is named by its text, a part of it, or #2.",
			syntax:   "stickypane todo <name> add|check|uncheck <item>",
			examples: []string{"stickypane todo plan add \"write tests\"", "stickypane todo plan check tests", "stickypane todo plan uncheck #1"},
			related:  []string{"card", "write", "checklist"},
		}},
		{[]string{"card"}, editCmd("card"), helpText{
			group: "notes", brief: "add or move a card of a board",
			what:     "Adds a card to a board note, or moves one to another column. A card is named by its text, a part of it, or #2; a missing note is made.",
			syntax:   "stickypane card <name> add|move <card> [--to <column>]",
			args:     "--to C  the column to add to, or to move to (needed to move)",
			examples: []string{"stickypane card work add \"login API\" --to Doing", "stickypane card work move login --to Done", "stickypane card work add \"docs\""},
			related:  []string{"todo", "board", "write"},
		}},
		{[]string{"chart"}, editCmd("chart"), helpText{
			group: "notes", brief: "set or add to a number of a chart",
			what:     "Sets a number of a chart note, or adds to it, and prints the new value. A negative number is a value.",
			syntax:   "stickypane chart <name> set|add <label> <number>",
			examples: []string{"stickypane chart tokens set input 1,200", "stickypane chart tokens add input 800", "stickypane chart tokens add delta -5"},
			related:  []string{"log", "kinds"},
		}},
		{[]string{"log"}, editCmd("log"), helpText{
			group: "notes", brief: "append a line to a log note",
			what:     "Appends a line to a log note; --time starts it with the time. After -- everything is words.",
			syntax:   "stickypane log <name> [--time] <text>",
			args:     "--time  start the line with the time",
			examples: []string{"stickypane log worklog \"tests passed\"", "stickypane log worklog --time \"deploy started\""},
			related:  []string{"say", "chart", "kinds"},
		}},
		{[]string{"say"}, editCmd("say"), helpText{
			group: "notes", brief: "add a message to a chat note",
			what:     "Adds a message to a chat note, said by agent unless --as names someone else.",
			syntax:   "stickypane say <name> [--as <who>] <text>",
			args:     "--as W  who says it (default: agent)",
			examples: []string{"stickypane say chat \"41/41 pass\" --as claude", "stickypane say chat \"thanks\""},
			related:  []string{"log", "watch"},
		}},
		{[]string{"set"}, editCmd("set"), helpText{
			group: "notes", brief: "change keys of a note or its place on the board",
			what:     "Changes one thing without reading or rewriting the note: key=value pairs for how it shows (open, size...) or its front matter (title). A missing note is made, and so is the board in a git repository without one.",
			syntax:   "stickypane set <name> key=value ...",
			examples: []string{"stickypane set plan open=true size=1/2", "stickypane set plan title=\"The plan\"", "stickypane set plan open=false"},
			related:  []string{"write", "show", "mv"},
		}},
		{[]string{"mv"}, manageCmd("mv"), helpText{
			group: "arrange", brief: "rename a note or move it into a tab or book",
			what:     "Renames a note, or moves it: <to> is a new name, a folder (deploy/) to move it into that tab or book, or . for the top level.",
			syntax:   "stickypane mv <name> <to>",
			examples: []string{"stickypane mv plan roadmap", "stickypane mv plan deploy/", "stickypane mv deploy/plan ."},
			related:  []string{"link", "archive", "rm"},
		}},
		{[]string{"link"}, manageCmd("link"), helpText{
			group: "arrange", brief: "show a file or folder of the project without copying",
			what:     "Shows a file or a folder of the project on the board without copying it (a symbolic link). A folder becomes a book.",
			syntax:   "stickypane link <path> [name]",
			examples: []string{"stickypane link README.md", "stickypane link docs guide", "stickypane link logs/app.log"},
			related:  []string{"show", "mv"},
		}},
		{[]string{"rm"}, manageCmd("rm"), helpText{
			group: "arrange", brief: "move a note, a page or a folder to .trash/",
			what:     "Moves a note, a page or a folder to .trash/. restore brings it back.",
			syntax:   "stickypane rm <name>",
			examples: []string{"stickypane rm plan", "stickypane rm deploy/notes"},
			related:  []string{"restore", "archive"},
		}},
		{[]string{"restore"}, manageCmd("restore"), helpText{
			group: "arrange", brief: "bring back what rm removed last",
			what:     "Brings back what rm removed last under that name.",
			syntax:   "stickypane restore <name>",
			examples: []string{"stickypane restore plan"},
			related:  []string{"rm", "archive"},
		}},
		{[]string{"archive"}, manageCmd("archive"), helpText{
			group: "arrange", brief: "move a note out of sight into archive/",
			what:     "Moves a note out of sight into archive/. It is kept; nothing is deleted.",
			syntax:   "stickypane archive <name>",
			examples: []string{"stickypane archive plan"},
			related:  []string{"hide", "rm", "restore"},
		}},
		{[]string{"answers"}, answersCmd(false), helpText{
			group: "agents", brief: "print what the user answered in a form",
			what:     "Prints what the user answered in a form note, as it stands.",
			syntax:   "stickypane answers <name> [--json]",
			args:     "--json  print the answers as JSON",
			examples: []string{"stickypane answers deploy", "stickypane answers deploy --json"},
			related:  []string{"wait", "form"},
		}},
		{[]string{"wait"}, answersCmd(true), helpText{
			group: "agents", brief: "wait for a form's button, then print the answers",
			what:     "Waits until the user presses a button of a form, then prints the answers. Exit code 3 when the time runs out.",
			syntax:   "stickypane wait <name> [--timeout 5m] [--json]",
			args:     "--timeout D  give up after this long, such as 5m (default: wait for good)\n--json       print the answers as JSON",
			examples: []string{"stickypane wait deploy --timeout 10m", "stickypane wait deploy --json"},
			related:  []string{"answers", "watch"},
		}},
		{[]string{"api"}, apiCmd, helpText{
			group: "scripts", brief: "(experimental) send the requests of a .http note",
			what:     "Lists the requests of a .http note, or sends one (by its name, a part of it or #2) or all, and prints the response, its checks and captures.",
			syntax:   "stickypane api <name> [request] [--env E] [--all]",
			args:     "--env E  the environment to use\n--all     send every request, in order",
			examples: []string{"stickypane api users", "stickypane api users \"Get user\" --env dev", "stickypane api users --all"},
			related:  []string{"kinds", "watch"},
		}},
		{[]string{"watch"}, watchCmd, helpText{
			group: "scripts", brief: "print what happens on the board as it happens",
			what:     "Prints what happens on the board as it happens: note.created, item.ticked, card.moved, form.submitted, log.appended, chart.changed... --exec runs a command for each, with STICKY_* set.",
			syntax:   "stickypane watch [--note N] [--type T] [--json] [--exec CMD] [--once] [--timeout 10m]",
			args:     "--note N     only these notes or folders, comma-separated\n--type T     only these events or their kinds (card, item.ticked)\n--json       print each event as JSON\n--exec CMD   run CMD for each event\n--once       end after the first event\n--timeout D  give up after this long, such as 10m",
			examples: []string{"stickypane watch --once --timeout 10m", "stickypane watch --note deploy --json", "stickypane watch --type card --exec \"echo moved\""},
			related:  []string{"wait", "mcp"},
		}},
		{[]string{"mcp"}, mcpCmd, helpText{
			group: "agents", brief: "serve the notes over MCP",
			what:     "Serves the notes over the Model Context Protocol on standard input and output, for agents that would rather call a tool.",
			syntax:   "stickypane mcp",
			examples: []string{"stickypane mcp"},
			related:  []string{"setup", "guide"},
		}},
		{[]string{"bridge"}, bridgeCmd, helpText{
			group: "scripts", brief: "serve the board to the desktop app",
			what:     "Not for typing: serves the board to the desktop app as JSON lines on standard input and output. It ends, and stops the scripts it started, when the app closes its input.",
			syntax:   "stickypane bridge",
			examples: []string{"stickypane bridge"},
			related:  []string{"start"},
		}},
	}
}

// lookup finds the command a word names.
func lookup(word string) (command, bool) {
	for _, c := range commands {
		for _, n := range c.names {
			if n == word {
				return c, true
			}
		}
	}
	return command{}, false
}

// initCmd makes the board and puts the agent guide where agents read it.
func initCmd(e env, args []string) error {
	fs := newFlags("init")
	noDocs := fs.Bool("no-agent-docs", false, "leave AGENTS.md and CLAUDE.md alone")
	skill := fs.Bool("skill", false, "install the guide as a Claude Code skill instead")
	if _, err := parse(fs, "init", args, 0, 0); err != nil {
		return err
	}
	if err := initcmd.Run(".", initcmd.Options{NoAgentDocs: *noDocs, Skill: *skill}, e.stdout); err != nil {
		return err
	}
	if dir, err := store.Resolve("."); err == nil {
		registry.Append(registry.Event{Type: "project.init", Project: filepath.Dir(dir)})
	}
	return nil
}

// changes are the commands that change a board; each one notes its project
// in the registry, once a minute, as soon as open has found it.
var changes = map[string]bool{
	"write": true, "todo": true, "card": true, "chart": true, "log": true,
	"say": true, "set": true, "mv": true, "link": true, "rm": true,
	"restore": true, "archive": true, "show": true, "hide": true,
}

// prune clears the registry of records nobody needs any more: the files of
// sessions whose process has died, events older than 90 days. It runs when
// a person opens the board, in the terminal or the desktop app: once a
// session, a few files listed, never on the commands an agent runs many
// times a minute, and before the app reads run/ to list live boards.
func prune() {
	registry.Prune(registry.Home(), time.Now())
}

// live records that this command works on the project here until it ends:
// defer live(kind)(). It does nothing where there is no board.
func live(kind string) (end func()) {
	dir, err := store.Resolve(".")
	if err != nil {
		return func() {}
	}
	return registry.Live(filepath.Dir(dir), kind)
}
