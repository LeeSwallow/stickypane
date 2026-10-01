# stickypane

A sticky-note board in your terminal, for you and your coding agent.

```
╔ 📌 Auth work ════════════════════════════════════════════════════════════════╗
║ To do (1)                Doing (1)                Done (1)                   ║
║ payments                 login API                schema                     ║
╚══════════════════════════════════════════════════════════════════════════════╝
╭ Login API ───────────────────────────╮╭ Auth design ─────────────────────────╮
│ ▓▓▓▓▓▓▓▓▓▓▓▓▓░░░░░░░ 2/3             ││ Decisions                            │
│ ☐ Write tests                        ││                                      │
╰──────────────────────────────────────╯│ Tokens live in a session cookie.     │
╭──────────────────────────────────────╮╰──────────────────────────────────────╯
│ check env before deploy              │╭ ● Work log ──────────────────────────╮
╰──────────────────────────────────────╯│ 14:02 tests passed                   │
                                        │ 14:10 started on review feedback     │
                                        ╰──────────────────────────────────────╯
n jot  enter open  a add  ? help  q quit
```

Your agent jots things down as plain Markdown files. stickypane shows them as
notes in a pane next to it: plain notes, kanban boards, checklists and logs.
Move a card or tick a box on the board and the change lands in the same file,
so the agent sees it too.

- **Nothing to configure.** No config file, no server, no MCP, no hooks.
- **Any agent.** If it can edit files, it can stick notes: Claude Code, Codex
  and others.
- **Any terminal.** A plain window, tmux, WezTerm, Zellij. stickypane is a
  visualization layer, not a terminal plugin.

## Install

```sh
brew install --cask LeeSwallow/tap/stickypane
# or
go install github.com/LeeSwallow/stickypane/cmd/stickypane@latest
```

macOS and Linux are supported.

## Quick start

```sh
cd your-project
stickypane init    # creates .stickypane/ and tells your agent about it
stickypane         # opens the board
```

`init` adds a short guide to `AGENTS.md` (or `CLAUDE.md` if you have one), so
your agent knows the board exists. Then ask it for anything: "keep a checklist
of this refactor on the board", "jot down what we decided".

Keep the board next to your agent:

```sh
tmux split-window -h stickypane             # tmux pane
tmux display-popup -E stickypane            # tmux popup
wezterm cli split-pane --right -- stickypane
```

## Notes are files

One Markdown file in `.stickypane/` is one note. Create a file to stick a
note, edit it to update the note, delete it to take the note down. A one-line
file is a complete note:

```markdown
check env before deploy
```

Front matter gives a note a shape. Every key is optional.

| Key     | Values                                           |
| ------- | ------------------------------------------------ |
| `type`  | `note` (default), `board`, `checklist`, `log`    |
| `title` | shown in the note's border                       |
| `color` | `yellow`, `pink`, `blue`, `green`, `purple`, `orange` |
| `pin`   | `true` keeps the note at the top                 |

**Board.** Each `## Heading` is a column and each top-level list item is a
card. Indented lines under a card are its details. Any other line under a
column shows up as a card too, and text before the first heading is the
board's description.

```markdown
---
type: board
title: Auth work
---
## To do
- payments
## Doing
- login API
  - refresh tokens come later
## Done
- schema
```

**Checklist.** `- [ ]` and `- [x]` lines, shown with a progress bar.

**Log.** One entry per line. The board shows the latest lines and follows
along as the file grows.

Notes are ordered by file name, pinned ones first. Prefix a number
(`10-plan.md`) to control the order. A file that does not fit its shape is
still shown, never hidden.

## Keys

| Board                 |                    | Open note |                         |
| --------------------- | ------------------ | --------- | ----------------------- |
| arrows, `h j k l`     | move focus         | `esc`     | close                   |
| `enter`               | open note          | `j` `k`   | move or scroll          |
| `n`                   | jot a note         | `G`       | jump to the end         |
| `a`                   | add by shape       | `H` `L`   | move a card sideways    |
| `p` / `c` / `R`       | pin, color, rename | `J` `K`   | reorder a card          |
| `x` / `D`             | archive, delete    | `space`   | tick a checklist item   |
| `e`                   | edit in `$EDITOR`  | `n`       | new card or item        |
| `?`                   | help               | `e`       | edit in `$EDITOR`       |

## How edits are written

The board never overwrites a file with what it last saw. An edit is an intent
("move this card to that column") applied to the file as it is at that moment,
and the file is then replaced in one step. If your agent changed that card in
the meantime, nothing is written and the board tells you. A note that is a
symlink is edited where it really lives.

Replacing the file has two limits:

- A program that keeps a note open and keeps appending to it
  (`some-command >> .stickypane/log.md`) goes on writing to the old file after
  you edit that note from the board. Writing a line at a time, the way agents
  do, is fine.
- A write that lands in the instant between the board reading a file and
  replacing it is lost.

## Adding a shape

A shape is one package under `internal/widget/` that implements the
`widget.Widget` interface, plus one line in `internal/kinds/kinds.go`.

## License

MIT
