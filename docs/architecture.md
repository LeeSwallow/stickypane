# Architecture

How the code is laid out, which way the layers depend on each other, and
how to add a shape of note. The principles the design is checked against
are in [VISION.md](../VISION.md).

## Layers

A package only imports packages of a lower layer. `internal/layers` checks
this on every test run: an import the wrong way, or a new package without a
layer, fails the build.

| Layer | Packages | Is |
| --- | --- | --- |
| 60 | `cmd/stickypane` | the program: one table of commands |
| 50 | `internal/app`, `internal/mcp` | the two front ends: the terminal UI and the MCP server |
| 40 | `internal/api`, `internal/initcmd` | what the command line and agents call; setting a project up |
| 30 | `internal/kinds`, `internal/arrange` | the registry of shapes; how a note is arranged |
| 20-22 | `internal/widget`, `internal/widget/<shape>` | the widget contract, then one package per shape |
| 10 | `internal/store` | the notes folder: files, sticky.json, watching |
| 5 | `internal/i18n`, `internal/httpfile` | the screen's words, in the language the locale asks for; reading and sending `.http` requests |
| 0 | `internal/doc`, `internal/theme`, `internal/env`, `internal/when`, `internal/layout`, `internal/editor` | leaves that know nothing of the board; `when` is how every time is written and read |

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

## A shape of note

Each shape is one package under `internal/widget/`. A shape with both a
reader and a writer is laid out the same way:

| File | Holds |
| --- | --- |
| `kind.go` | the `widget.Kind`: name, icon, keys, blurb, example, command, usage |
| `format.go` | the file's grammar, shared by the reader and the writer |
| `model.go` | the reader: what the file says, with no screen state in it |
| `ops.go` | the writer: the `doc.Op` intents the screen and the command line apply |
| `view.go` | the screen: drawing, keys, clicks, the cursor |
| `events.go` | what happened between two versions of the file, for `stickypane watch` |

Small shapes (`note`, `logview`, `script`) stay in one file.

To add a shape:

1. Make `internal/widget/<shape>/` with the files above. The widget
   implements `widget.Widget`; `widget.Clicker` if it takes the mouse.
2. Fill in its `Kind`, including `Command` and `Usage`: `stickypane kinds`
   prints them and the agent reads them there.
3. Add one line to `internal/kinds.Default`.
4. If the command line should change one thing in it, add a function to
   `internal/api/edit.go` that applies one of its ops, and a command to the
   table in `cmd/stickypane/commands.go`.
5. Describe it in `skills/using-the-board/references/kinds.md`.

`TestEveryKindKeepsTheContract` in `internal/kinds` runs against every
registered shape: its catalog text, drawing within every width, keys and
an empty file. A new shape that passes it works on the board.

## The command line

`cmd/stickypane` follows the shape of Go's own `cmd/go`, with the standard
library only:

| File | Holds |
| --- | --- |
| `main.go` | `main`, signal handling, opening the board |
| `commands.go` | the table of commands and `env` (context and streams) |
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
