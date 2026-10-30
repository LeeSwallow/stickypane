# Changelog

All notable changes are listed here. The format follows
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/); the release notes
on GitHub are built from pull request labels.

## [Unreleased]

### Changed

- A note keeps the width and height chosen for it when other notes share
  the screen, on the terminal and in the app. A row whose sized notes do
  not add up to the whole width leaves the rest empty, and only notes with
  no size of their own take what a row leaves. A note is as tall as its
  `rows`, or its content, instead of as tall as the tallest in its row, and
  the screen is no longer padded out: the notes are squeezed only when they
  do not fit. The app's `board.load` says which notes are `sized` (#65).

- The agent plugin is in one folder, `plugin/`: its manifests, skills,
  commands, rules, hooks, hook scripts and the agent guide's head. The
  marketplaces `stickypane` (`.claude-plugin/marketplace.json` and
  `.agents/plugins/marketplace.json`, still at the root) point at
  `./plugin`, and the plugin is still `board`, so an install from
  `LeeSwallow/stickypane` updates as before, with no step to take;
  `scripts/` keeps only the scripts for developing stickypane (#52).

- A `spark` chart is a line chart that fills the pane (min and max on an
  axis, first and last labels, the latest value marked), and a `heat` chart
  a calendar with weeks two to four cells wide, month labels and a scale.
  Zoomed, they use the whole screen (#11).

### Added

- `stickypane help <command>` and `help <shape>` print one command (arguments,
  examples, related) or a note's format; `help start` is the way in, shown
  too where there is no board; `help` is a short overview by task and
  `help --all` the whole page. Help speaks the screen's language, Korean
  included (#17).
- An `html` note: `.html` and `.htm` files in `.sticky/` are shown as text
  in the terminal, and cannot be changed from a screen.
- The desktop app (#10), built here and not yet released: stickypane in a
  window, for watching the board and answering it. It comes with
  `stickypane` inside it, so the app is the one install.
  - Notes drawn in full: Markdown with tables, code, Mermaid and maths,
    every shape, and `.html` pages in a sandbox that reaches no network,
    no storage and nothing of the app (#18, #20).
  - Clicks become the same changes as the terminal's keys: tick, answer a
    form, drag a card, run a script or send a request after a question;
    **+ New note** makes any shape from its options; a note's ⋯ menu
    moves, archives or deletes it, with undo (#18).
  - A free canvas beside the grid, and a note torn off into a small window
    of its own that stays on top. Places are kept per computer in
    `~/.stickypane/overlay.json`; the notes do not change (#19).
  - What an agent changed moves once: a card carried to its column, new
    notes dropping in, a stalled card's wobble. Settings → Motion: system,
    less or off (#21).
  - A home window that lists this computer's boards by day, a tray icon,
    a hotkey (⌥Space) and windows kept on top (#24).
  - Settings → Terminal and agents installs the app's `stickypane` as the
    terminal command (`~/.local/bin`, or `/usr/local/bin` with a password)
    and connects Claude Code and Codex through its `stickypane setup`,
    showing the plan first. A folder without a board offers 보드 만들기
    (#22).
  - `stickypane init --view app` makes a plain `stickypane` open the board
    in the app, and in the terminal when the app is missing (#22).
  - Packaged as a DMG drawn on graph paper (drag to Applications) and a
    Linux tar.gz with a `.desktop` entry and `install.sh` (#23). The DMG
    comes with the first app release.
- `stickypane bridge`: serves the board as JSON lines on standard input and
  output, for the desktop app. It is not for agents. `themes.list` gives
  every theme's colours.
- `stickypane setup --command <path>` registers the MCP server by a full
  path, and `stickypane init` refuses the home folder and the root (#22).
- `~/.stickypane` (or `$STICKYPANE_HOME`), written only when the folder
  already exists: `events/YYYY-MM-DD.jsonl` (lines of `{v: 1, at, type, …}`) and
  `run/<pid>.json` record
  project paths, command names, times and process ids, so the desktop app
  can list your boards. See the README.
- Opening a board, in the terminal or with `stickypane bridge`, prunes
  `~/.stickypane`: run files of sessions whose process has died, events
  older than 90 days, touch markers a day old. stickypane is the only
  program that deletes these files (#32).
- `view` and `style` keys in `sticky.json`, for the desktop app; the
  terminal ignores them.
- A note's height from the board: `shift+up` and `shift+down` (with `pgup`
  and `pgdn`, five lines), `0` for the height of its content, and the mouse:
  drag a note's right edge for its width, its bottom edge for its height,
  and double-click the bottom edge for the height of its content.

### Changed

- A note's `size` is a share of the screen: `1/4`, `1/3`, `1/2`, `2/3`,
  `3/4` or `1`, which `+` and `-` step through and the border names for a
  moment. `card`, `half` and `page` are still read, and written as a
  fraction the next time the size changes; `stickypane set` and `write
  --size` take both and write `half` and `page` as `1/2` and `1`.
- Tiling: a note gets at least six lines (or what it needs) before the next
  screen, not ten; a row's notes share the cells left by rounding; and a
  note with a height of its own keeps it while the others share what is
  left.
- `0` (or a double click on the bottom edge) on a note whose front matter
  or kind gives it a height (a log) writes `"rows": -1` to `sticky.json`:
  as tall as its content, over that height.
- Older versions of stickypane do not know the fractions or `rows: -1`: a
  `sticky.json` written by this version shows its notes there at their
  default size and height.

### Fixed

- The welcome note of a new board says how to start in the app as well as
  in the terminal (#25).
- The session hook and `/board:status` take only `.sticky/` for a board,
  not `~/.stickypane` (#26).
- `stickypane setup` makes no board and writes no `AGENTS.md`; it says to
  run `stickypane init` in a project. It says when no agent was found, and
  the README says what `--undo` removes (#26).
- `stickypane mcp` starts in a project without a board: a tool that
  changes the board makes it in a git repository, and any other tool says
  to run `stickypane init` (#26).
- The plugin manifests say 0.1.1 (#26).

## [0.1.1] - 2026-10-10

### Fixed

- Windows: a script's output reaches its log; a skill or an instruction
  file with CRLF line ends is read as with LF; text files keep LF line ends
  on a Windows checkout.
- `stickypane version` names the module version of a `go install` build,
  not `dev`.
- The README's first command example works in a new repository.

### Added

- `brew install LeeSwallow/tap/stickypane`: a Homebrew cask in a tap.

## [0.1.0] - 2026-10-08

The first version. stickypane shows the files of a project's `.sticky/`
folder as a board in a terminal pane, for a coding agent and its user to
share.

### Added

- Notes as plain files: Markdown notes, and by front matter kanban boards,
  checklists, charts (bar, spark, heat), forms and chats; `.log` files
  followed live; `.sh` and `.ps1` scripts with a Run button; `.http`
  requests (experimental); folders as tabs and books; Mermaid diagrams.
- Times kept by the board, not the agent: when a note was made, an item
  ticked, a card moved, a message said; shown in the user's locale.
- Notes that meet: `[[links]]` followed with `f`, `![[embeds]]` drawn in
  place, charts computed `from:` other notes, and events (`stickypane
  watch`, MCP `wait_event`) that a command can react to.
- An index of every note (`i` on the board, `stickypane index`, MCP
  `index`), and a settings panel (`S`, `stickypane config`) where every
  setting has a default: theme, language, time format, editor, staleness.
- A screen of fixed panes that redraws when something changes, not on a
  tick; keys and mouse; a built-in vi-like editor, or any editor, terminal
  or GUI; ten color themes.
- A command line and an MCP server for agents: `show`, one-thing edits
  (`todo`, `card`, `chart`, `log`, `say`, `set`), `rm`/`mv`/`restore`/
  `archive`/`link`, forms with `answers` and `wait`.
- A Claude Code and Codex plugin in layers: commands that name a skill,
  five skills (the way in, tracking progress, asking the user, talking in
  a chat, connecting notes) and the rules they share. The program carries
  the same skills for MCP clients, as prompts and `guide` topics.
- macOS, Linux and Windows; tmux, WezTerm, Zellij and Windows Terminal
  panes; sh and PowerShell.

### Known limits

A program that keeps a log file open and appends to it stops being followed
after the board edits that file; no counts or visual mode in the editor.
