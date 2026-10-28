# Architecture

How the code is laid out, which way the layers depend on each other, and
how to add a shape of note. The principles the design is checked against
are in [VISION.md](../VISION.md).

## Layers

A package only imports packages of a lower layer. `internal/layers` checks
this on every test run: an import the wrong way, or a new package without a
layer, fails the build. It also fails on a time layout outside
`internal/when` (a string such as `"15:04"`, or `time.RFC3339`): a time is
written through a named form in `when`, and a clock is handed in as a
`now func() time.Time`, never read from `time.Now` where a stamp is made.

| Layer | Packages | Is |
| --- | --- | --- |
| 60 | `cmd/stickypane` | the program: one table of commands |
| 50 | `internal/app`, `internal/mcp`, `internal/bridge` | the front ends: the terminal UI, the MCP server, and the bridge the desktop app talks to over standard input/output |
| 40 | `internal/api`, `internal/initcmd` | the use cases every front end goes through: the terminal, the bridge, MCP and the command line; setting a project up |
| 30 | `internal/kinds`, `internal/arrange` | the registry of shapes; how a note is arranged |
| 20-22 | `internal/widget`, `internal/widget/<shape>` | the widget contract, then one package per shape |
| 12 | `internal/runner` | running a script and keeping its log, for every front end |
| 10 | `internal/store` | the notes folder: files, sticky.json, watching |
| 5 | `internal/harness` | the plugin's skills, read for MCP and `stickypane guide` |
| 5 | `internal/i18n`, `internal/httpfile` | the screen's words, in the language the locale asks for; reading and sending `.http` requests |
| 5 | `internal/registry` | the record of which boards are used on this machine, kept in `~/.stickypane` when the desktop app has made it: written, read and pruned only here |
| 3 | `internal/rpc` | gRPC calls on the standard library's HTTP/2; `rpc/rpctest` is a server for tests (2) |
| 0 | `internal/wsock`, `internal/proto` | a WebSocket client and server; protocol buffers without generated code, with `proto/prototest` (1) for tests |
| 0 | `.` (the module root) | the plugin's files (`skills/`, `rules/`, `commands/`), embedded |
| 0 | `internal/atomicfile` | the one way a file is replaced or made: a synced temporary file renamed (or linked) into place, so no reader sees half of it |
| 0 | `internal/doc`, `internal/theme`, `internal/env`, `internal/when`, `internal/markdown`, `internal/lineedit`, `internal/layout`, `internal/editor` | leaves that know nothing of the board; `when` is how every time is written and read |
| 0 | `internal/httpfile/syntax` | the grammar of a `.http` file, read without the code that sends it: the API note draws from this alone |

The packages stay side by side under `internal/` rather than in folders by
role. The layers do not split by role: widgets draw with the theme, the
arrangement asks a widget's kind for its default size, and the API applies
the widgets' writers. Folders by role would point the wrong way; the
table above, kept true by a test, is the hierarchy.

`internal/` is a Go rule, not a web habit: nothing outside this module can
import these packages, so none of them is a public API and all of them can
change while the design does.

## Where it runs

`internal/env` is the one place that knows the difference between macOS,
Linux and Windows, between shells, and between the tools that can open a
pane next to the agent. It detects, from the environment alone:

| What | How | Used for |
| --- | --- | --- |
| pane tool | `TMUX`, `ZELLIJ`, `WEZTERM_PANE` or `TERM_PROGRAM`, `WT_SESSION`, in that order | the split command init and `stickypane env` suggest |
| shell | `SHELL`; on Windows, whether `PSModulePath` holds the user's PowerShell modules | how to set a variable, in messages |
| locale | `STICKYPANE_LANG`, `LC_ALL`, `LC_MESSAGES`, `LANG`, then the system: macOS `AppleLocale`, Windows `GetUserDefaultLocaleName` | the screen's language |
| interpreters | `sh` for `.sh`, `pwsh` or `powershell` for `.ps1` | running a script note |
| editor | `VISUAL`, `EDITOR`, else `vi` or `notepad` | `E` on the board |

What is PowerShell's own is in `powershell.go`, and each system's way of
naming its locale in `locale_darwin.go`, `locale_windows.go` and
`locale_other.go`. `stickypane env` prints what was found. A new pane tool
is one entry in the table in `pane.go`.

## What is ours and what is not

The board keeps few dependencies, and writes the parts it uses little of:

| Ours | Instead of | Why |
| --- | --- | --- |
| `internal/markdown` | glamour, goldmark, chroma, bluemonday | notes use a small part of Markdown; the renderer and its HTML, emoji and syntax-highlighting libraries were most of the binary |
| `internal/lineedit` | bubbles' textinput | one line of text on the bottom line |
| `internal/wsock`, `internal/proto`, `internal/rpc` | gorilla/websocket, google.golang.org/protobuf, grpc-go | the API note sends a script of messages and one call at a time; the libraries would be most of the binary again |
| `internal/when`, `internal/env` | | how times and the environment are read is the board's own decision |

Kept: bubbletea, lipgloss and x/ansi (the terminal itself), fsnotify
(following the folder on every system) and mermaid-ascii (laying out
diagrams is its own craft, and it costs a quarter of a megabyte).

## Reading and writing

**Reading.** `store.Load` reads the board in one pass: every `sticky.json`
once, into a `store.Settings` snapshot, then the notes of every tab. A note
whose file kept its size and time is handed back as it was read last time,
so a reload after one write reads one file. The screen keeps the widget of
a note that did not change, and what it drew.

**Writing.** Every change to a note is a `doc.Op`, an intent such as "move
this card to that column", applied by `store.Apply` to the file as it is on
disk now and written in one step. The screen, the command line and MCP all
write this way, so an agent and the user editing the same note do not
overwrite each other. Settings are written the same way, through one read,
change and replace step.

**One place for each rule.** `internal/api` is the use-case layer: the
terminal (`internal/app`), the bridge and the command line change the
board through it, never through the store. A note is made by
`api.CreateFrom` (named and stamped on the API's clock), tended by
`api.TendAll`, changed by a key's op through `api.Do`, saved from the
editor by `api.Save`, archived, trashed, restored and moved by
`api.Archive`, `api.Remove`, `api.Restore`/`api.Return` and `api.Move`,
and a `.http` request is sent by `api.SendPart`. Settings are changed by
`api.ChangeSettings`, over `prefs.Normal`, the one rule for what each
setting takes. api answers with values (`api.Moved`, `api.Sent`); the
front end says them in its own words. A note's colour is
`arrange.ColorIndex`, and the language a choice comes to `i18n.Resolve`.
What a front end still takes from the store only reads it, or arranges
the screen in `sticky.json` (a view, the tab).

**Events.** Each kind may say what happened in a note between two
versions (`Kind.Events`: item.ticked, card.moved, form.submitted...).
`api.Changes` compares two readings of the board and adds note.created,
note.removed and note.changed; `api.Watch` follows the folder and hands
the events to `stickypane watch` and the MCP tool `wait_event`. Kinds may
also fill in what is missing by themselves (`Kind.Tend`: the time an item
was ticked or a card moved), and say when they next change as time passes
(`widget.Timed`), so the screen wakes then instead of ticking.

**Arranging.** Whether a note is open, its size, height, pin, color and
title come from three places, in order: the user's choice in `sticky.json`,
the note's front matter, the note's kind. `internal/arrange` is the only
code that decides; the screen and `stickypane list` and `set` ask it.

**The registry.** `~/.stickypane` has one writer, `internal/registry`:
`registry.Event` is a line of `events/<date>.jsonl`, `registry.Run` the
file `run/<pid>.json`, and a `registry.Writer` writes them on its own
clock, through `internal/when`. `registry.Read` reads them back for the
desktop app, and `registry.Prune`, run when the board or the bridge
opens, is the only code that deletes them. `internal/registry/testdata/home`
is a home written with a fixed clock; a test keeps the writer to it, and
the app's tests read it rather than copy the format.

## A shape of note

Each shape is one package under `internal/widget/`. A shape with both a
reader and a writer is laid out the same way:

| File | Holds |
| --- | --- |
| `kind.go` | the `widget.Kind`: name, icon, keys, blurb, example, command, usage; the create options and `Compose`, which writes a new file from them |
| `format.go` | the file's grammar, shared by the reader and the writer |
| `model.go` | the reader: what the file says, with no screen state in it |
| `ops.go` | the writer: the `doc.Op` intents the screen and the command line apply |
| `view.go` | the screen: drawing, keys, clicks, the cursor |
| `events.go` | what happened between two versions of the file, for `stickypane watch` |

Small shapes (`note`, `logview`, `script`) stay in one file. `html` is view-only, so its `ops.go` and `events.go` hold only a comment.
Only `view.go` imports lipgloss or bubbletea, and no widget imports the
network, `os/exec` or `internal/httpfile` (the API note reads its file with
`internal/httpfile/syntax`); `internal/layers` checks both.

**The Env.** A widget keeps no package variables. What it reads that is not
in its file comes in a `widget.Env`, handed to `Kind.Parse`, `Kind.Data` and
`Kind.Act`: the clock, the translator, the theme's styles, how long a card
may stall, how a time is written (`when.Formatter`) and how to find the
note a `![[name]]` line shows. The screen makes one from its own clock and
settings and makes its widgets again when it changes; the command line and
the bridge use the API's (`api.NewWithEnv`, `API.Env`), so what an action
writes carries the API's clock. Tests hand a widget an Env with a stopped
clock and run in parallel.

To add a shape:

1. Make `internal/widget/<shape>/` with the files above. The widget
   implements `widget.Widget`; `widget.Clicker` if it takes the mouse.
2. Fill in its `Kind`, including `Command` and `Usage`: `stickypane kinds`
   prints them and the agent reads them there.
3. Give it `Options` and `Compose` if a screen should make one from a
   form: what to ask (text, lines, choice, number, bool), and the whole
   new file written from the answers in the shape's own format.
   `Kind.Make` checks the answers and fills in the defaults; the bridge's
   `kinds.list` and `note.create` and `stickypane write --opt` use it, so
   no screen writes a shape's format itself. An option's label is a key
   of the "labels" in `internal/i18n/translations`.
4. Add one line to `internal/kinds.Default`.
5. If the command line should change one thing in it, add a function to
   `internal/api/edit.go` that applies one of its ops, and a command to the
   table in `cmd/stickypane/commands.go`.
6. Describe it in `skills/using-the-board/references/kinds.md`, and in a
   line of `rules/writing-notes.md` if agents write it by hand.

`TestEveryKindKeepsTheContract` in `internal/kinds` runs against every
registered shape: its catalog text, drawing within every width, keys and
an empty file. A new shape that passes it works on the board.
`TestEveryKindComposesFromItsOptions` holds its options to the same:
well formed, translated, and a note made with the defaults reads back as
the same shape.

## The plugin

The repository is also the Claude Code and Codex plugin, in four layers,
each pointing only at the one below:

| Layer | Folder | Is |
| --- | --- | --- |
| commands | `commands/` | `/board:status`, `/board:track`, `/board:ask`, `/board:say`, `/board:setup`: a line that names a skill |
| skills | `skills/<name>/SKILL.md` | one job each, starting with the commands to run; `using-the-board` is the way in and names the others |
| references | `skills/<name>/references/` | read when the skill points there: the shapes, in a paragraph each |
| rules | `rules/` | each rule once: the commands, what the board writes itself, how to write and word a note; the agent guide is made of them |

The program embeds `skills/`, `rules/` and `commands/` (`harness.go` at the
module root), and `internal/harness` hands the skills to the MCP server, as
prompts and as topics of the `guide` tool, and to `stickypane guide`;
`internal/initcmd` builds the agent guide from `rules/`. The
tests in `internal/initcmd` keep the layers pointing at each other: every
command names a skill that exists, and the way in names every skill.

## The command line

`cmd/stickypane` follows the shape of Go's own `cmd/go`, with the standard
library only:

| File | Holds |
| --- | --- |
| `main.go` | `main`, signal handling, opening the board |
| `commands.go` | the table of commands, each with its help, and `env` (context and streams) |
| `help.go` | `help`: the overview, one command or shape, the way to start; words from `internal/i18n` |
| `errors.go` | `exitCode`: the one place errors become exit codes 0, 1, 2, 3, 130 |
| `flags.go` | `parse`: flags anywhere among the words |
| `notes.go`, `edit.go`, `settings.go`, `kinds.go` | the commands |

A command is `func(env, []string) error`. Its usage is the lines of the
usage text that start with its name, so help and the table cannot drift.
Tests call `run` with buffers; nothing reads globals.

## Measuring

`go test ./internal/store ./internal/app -run XXX -bench .` measures a
reload and a redraw of a busy board. Run it before and after a change that
touches either.
