# Working with agents

How an agent learns about the board, and the commands and MCP tools it
uses. Back to the [README](../README.md).

- [Teach the agent, once](#teach-the-agent-once)
- [The plugin](#the-plugin)
- [The guide in CLAUDE.md or AGENTS.md](#the-guide-in-claudemd-or-agentsmd)
- [Commands for agents](#commands-for-agents)
- [MCP](#mcp)
- [The bridge](#the-bridge)

Editing the files is all an agent needs. The commands and MCP tools are
for scripts, and for agents that would rather call a tool than read a
note first.

## Teach the agent, once

| Way | How | For |
| --- | --- | --- |
| the plugin | `stickypane setup` | Claude Code and Codex: five skills, slash commands, a session hook |
| a guide in its instructions | `stickypane init` writes a short guide into `CLAUDE.md` or `AGENTS.md` | any agent that reads those files |
| MCP | `stickypane setup --mcp`, or `claude mcp add stickypane -- stickypane mcp` | any MCP client |

## The plugin

`stickypane setup` installs the plugin for every agent it finds, through
the agents' own commands. It makes no board and writes no instruction
file, so it is safe to run from any folder; `stickypane init` in a project
makes its board, and in a project that already has one takes out a guide
the plugin now teaches.

```sh
stickypane setup                       # asks, then installs for Claude Code and Codex
stickypane setup --scope project --mcp --yes
stickypane setup --undo                # uninstalls the plugin again
```

`--scope` is where Claude Code installs it: `user` (every project, the
default), `project` (this repository, shared with the team) or `local`
(this repository, just you). `--mcp` also registers `stickypane mcp` as an
MCP server, `--command` gives the full path of the stickypane that server
runs (the desktop app passes its own copy), `--agents claude,codex` limits
it to some agents, and `--dry-run` prints the commands without running
them. `--undo`
uninstalls the plugin from each agent found (from Claude Code at the
`--scope` given, `user` unless you say); with `--mcp` it also removes the
`stickypane` MCP server. It leaves the marketplace entry, the boards and
any guide `init` wrote in place. In a terminal it
asks for what the flags did not say.

The plugin is in layers. The way in, `using-the-board`, says which of four
skills fits a job: `tracking-progress`, `asking-the-user`,
`talking-in-chat` and `connecting-notes`. The commands `/board:show`,
`/board:status`, `/board:index`, `/board:track`, `/board:ask`,
`/board:say`, `/board:kinds` and `/board:setup` each start one. A session
hook tells the agent what is on the board when a session starts in a
project that has one. In a git repository the agent's first
`stickypane show` or `stickypane todo` makes the board, so no setup step
is needed.

[plugin.md](plugin.md) is for changing the plugin itself.

## The guide in CLAUDE.md or AGENTS.md

`stickypane init`, and the first run of `stickypane` in a git repository,
puts a short guide where the agent reads it: `CLAUDE.md` and `AGENTS.md`
when they exist, a Claude Code skill (`.claude/skills/stickypane/SKILL.md`)
when the project has a `.claude` folder but no `CLAUDE.md`, and a new
`AGENTS.md` when there is nothing. An agent with the plugin installed gets
nothing in its file, since the plugin teaches it. `init` works outside a
git repository too, but not in the home folder or the root, where every
project below would find the board; it is safe to run again. `--view app` (or `both`)
shows the board in the desktop app, and `--style neat` gives the app its
neat look; both go to `sticky.json`. With `app`, a plain `stickypane` in
the project opens the board in the app (or in the terminal, saying how to
get the app, when it is not installed); with `both`, in the app and the
terminal. `--skill` writes the skill
instead of the instruction files and `--no-agent-docs` makes the folder
only. `stickypane guide` prints the guide; it stays English.

## Commands for agents

Put something in front of the user:

```sh
stickypane show plan                 # a note, by name
stickypane show README.md            # any file of the project: linked and shown
stickypane show logs/app.log         # a log file: linked, shown and followed live
stickypane show docs/                # a folder: linked as a tab of its own
stickypane hide plan
```

Read and write whole notes, and read the user's answers:

```sh
stickypane list --json                       # every note: name, title, type, open, size
stickypane cat plan                          # print a note's file
echo "- [ ] build" | stickypane write plan --type checklist --title "Release" --open
stickypane answers deploy                    # what the user answered in a form
stickypane wait deploy --timeout 10m         # the same, once a button is pressed
```

`wait` exits with code 3 when its time runs out. Both take `--json`.
`write` replaces the whole note, exactly as overwriting the file would, so
read a note before rewriting it. A name may point into a book
(`docs/intro`) or at a log (`build.log`). With `--opt`, `write` reads
nothing from standard input: the shape writes the note from its create
options, the same ones the desktop app's create form shows
(`stickypane kinds board` lists them):

```sh
stickypane write sprint --type board --opt columns=Backlog --opt columns=Done --opt "cards=Backlog: login API @codex-1"
```

**Small changes without reading the note.** An agent that ticks an item by
rewriting the whole checklist has to read it first, and may write it back
wrong. These commands change one thing and print where the note stands:

```sh
stickypane todo plan add "write tests"       # plan.md: 0/1
stickypane todo plan check tests             # plan.md: 1/1
stickypane card work add "login API" --to Doing
stickypane card work move login --to Done    # work.md: 2 cards
stickypane chart tokens set input 1,200      # tokens.md: input = 1,200
stickypane chart tokens add input 800        # tokens.md: input = 2,000
stickypane log worklog --time "tests passed" # worklog.md: 12 lines
stickypane say chat "41/41 pass" --as claude # chat.md: 3 messages
stickypane set plan open=true size=1/2       # to sticky.json
stickypane set plan title="The plan"         # to the note's front matter
```

A note that does not exist yet is made, open on the screen, so an agent
never has to know the file format for these. An item, a card or a column is
named by its text, by a part of it that nothing else has, or by its
position (`#2`). When the name fits nothing, or more than one thing, the
error lists what there is.

To wait for something other than a form, an agent uses
`stickypane watch --once` ([usage.md](usage.md#notes-that-react)).

## MCP

`stickypane mcp` serves the notes over the Model Context Protocol on
standard input and output:

```sh
claude mcp add stickypane -- stickypane mcp
```

For another MCP client:

```json
{ "mcpServers": { "stickypane": { "command": "stickypane", "args": ["mcp"] } } }
```

It is layered the way the plugin is, so an agent finds its way without
reading everything:

| Layer | Over MCP | At the command line |
| --- | --- | --- |
| the way in | the `guide` tool | `stickypane guide` |
| a skill for a job: tracking progress, asking the user, talking in a chat, connecting notes | `guide` with the skill as `topic`, or the prompt of that name | `stickypane guide tracking-progress` |
| one thing changed | `index`, `read_note`, `write_note`, `todo`, `card`, `chart`, `log`, `say`, `set_keys`, `arrange_note` (show, hide, move, remove, restore) | the commands above |
| waiting for the user | `read_answers`, `wait_event` | `stickypane wait`, `stickypane watch` |

The skills are the plugin's own files, carried in the program, so a client
without the plugin reads the same ones. `write_note`, like `write`, replaces
the whole note.

## The bridge

`stickypane bridge` is not for agents or for typing: it serves the board as
JSON lines on standard input and output to the desktop app, which is in
progress (#10). It ends, and stops the scripts it started, when its input
closes.
