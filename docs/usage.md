# Using the board

Keys, the mouse, the editor, the index, moving notes, and watching what
happens. Every command has its own help in the program: `stickypane help
<command>` prints one with its examples, and `stickypane help --all`
prints every command with every option on one page. Back to the
[README](../README.md).

- [Keys](#keys) and [how the screen is shared](#how-the-screen-is-shared)
- [Mouse](#mouse)
- [Editing a note](#editing-a-note)
- [The index](#the-index)
- [Deleting and moving](#deleting-and-moving)
- [Notes that react](#notes-that-react): `stickypane watch`
- [How edits are written](#how-edits-are-written)

For the commands an agent uses, see [agents.md](agents.md). For the shapes
of a note, [shapes.md](shapes.md).

## Keys

The board's keys are vim's where vim has one.

| Any note | | Open kanban or checklist | |
| --- | --- | --- | --- |
| `h` `j` `k` `l` | the note on that side | `←` `→` (zoomed: `h` `l`) | change column |
| `gg` / `G` | first, last note | `↑` `↓` (zoomed: `k` `j`) | change card or item |
| `tab` / `shift+tab` | next, previous note | `H` `L`, zoomed | move a card sideways |
| `enter` | open, then zoom | `J` `K`, zoomed | reorder a card |
| `o` | open or close | `space` | tick an item |
| `>` / `<` (`+` `-`) | wider, narrower | `n` | new card or item |
| `shift+up` / `down` | taller, shorter (`pgup` `pgdn`: by five) | | |
| `0` | as tall as its content | **Open script** | |
| `ctrl+e` / `ctrl+y` (`↓` `↑`) | scroll a line | `enter` | run, after a yes |
| `ctrl+d` / `ctrl+u` | scroll half a page | | |
| `pgdn` / `pgup` | scroll a page | **Open form** | |
| `N` | jot a note | `↑` `↓` (zoomed: `k` `j`) | change control |
| `a` | add by shape | `enter` / `space` | choose, type, press |
| `p` / `c` / `R` | pin, color, rename | | |
| `H` `L` | move earlier, later | **Zoomed note** | |
| `K` `J` | move a row up, down | `esc` | back |
| `,` / `.` | turn a book's pages | `j` `k` `g` `G` | scroll |
| `m` | move to a folder | | |
| `x` / `dd` / `u` | archive, delete, undo | | |
| `yy` | copy the note's path | | |
| `e` / `E` | edit here, in `$EDITOR` | | |
| `r` / `?` / `q` | reload, help, quit | | |
| `T` | next theme | | |
| `z` | zoom | | |
| `[` / `]` | previous, next screen | | |
| `1`–`9` / `gt` `gT` | switch tab | | |
| `i` / `/` | the index | `S` | settings |
| `f` | linked notes | | |

On the board `h` `j` `k` `l` always move between notes, as the screen shows
them: down from the last row goes on to the next screen, and from a closed
note they go along the title bar. `H` `J` `K` `L` and `gg` `G` are the
board's too. The focused open note takes its other keys where it is, so you
tick an item, add a card or move the cursor with the arrows without
zooming. Zoomed (`enter`, or `z` for a note that uses `enter` itself, such
as a form), a note takes all of its keys: a kanban moves its cursor with
`h` `j` `k` `l` and its cards with `H` `J` `K` `L`, and a note without a
cursor scrolls with `j` `k` `g` `G`. A note with a cursor scrolls to keep
the cursor in view.

`yy` copies where the focused note's file is, from the project
(`.sticky/plan.md`, or the project's own file a linked note shows), through
the terminal (OSC 52): it works over ssh, and in tmux with
`set -g set-clipboard on`.

A count before a key repeats it: `3J` moves a note three rows down, `3l`
goes three notes right. A digit alone goes to its tab once no key follows
it. With the Korean 2-set layout on, the board reads `ㅓ` as `j` and every
other jamo as the key it is on.

These are the keys as they come. Each board key belongs to a named action
(`note-archive`, `jot`), and `stickypane keys` lists them with the keys
they have now; [configuration.md](configuration.md#keys) says how to
change them.

### The command line

`:` opens vim's command line on the bottom line: type a command, `enter`
runs it, `esc` drops it. `↑` `↓` bring back what you ran this session, and
`tab` completes a command's name, a setting, a theme or a note. A command
may be shortened to any start no other shares: `:col 3`, `:th nord`.

| Command | |
| --- | --- |
| `:q` | quit (`:q!` and `:wq` too) |
| `:w` | nothing to write: the board saves as you go |
| `:set [setting [value]]` | list the settings, show one, or change one (`:set stale=1h`) |
| `:theme` / `:col` / `:lang [value]` | show or change that setting |
| `:e <note>` | zoom a note by its name or title, in any tab; alone, the index |
| `:new [shape] [title]` | add a note: `:new idea`, `:new checklist Release`; alone, the shapes |
| `:jot [text]` | jot a note |
| `:tab <n\|name>` | go to a tab |
| `:archive` / `:delete` / `:undo` | `x`, `dd`, `u` by name (`:restore` is `:undo`) |
| `:3` | focus the third note of the tab |
| `:keys` / `:help [command]` | the keys; what a command does |

A command does what its key or the settings panel does, and writes
through the same code. `:` is the `command-line` action, so
`stickypane keys set command-line ";"` moves it. On the line a jamo is
typed as a jamo.

### How the screen is shared

The board is three columns wide unless the `columns` setting says
otherwise (`1` to `4`; `S` on the board, or `stickypane config columns 4`).
Notes go left to right, a row at a time. A note without a size of its own
takes one column, and the notes of a row share its width equally, so a
last row of two notes on three columns is two halves. A note whose size you
chose (with `>` `<`, a drag, `stickypane set` or its front matter) spans its
share of the columns, rounded to the nearest: on three columns `1/2` and
`2/3` are two, `1/4` and `1/3` one. The other notes of its row share what
is left. On a screen too narrow for a column of 20 cells, fewer columns are
drawn; nothing is written. `columns auto` brings back the layout of before,
where the sizes alone decide how many notes fit in a row.

A note is as tall as the `rows` it was given, or else as its content; notes
in a row start at its top, and the next row starts below the tallest. Only
when the notes do not fit are the tall ones squeezed, sharing what the
short ones leave, and the screen never scrolls as a whole. A note gets at
least six lines, or all it needs if that is less; notes that would get
less go to the next screen, shown as `2/3` under the title bar.

`>` and `<` widen or narrow a note by one column and its border says the
new size for a moment; `H` `L` move it a place, `K` `J` a row up or down,
trading places with the note there. `shift+up` and `shift+down` make it a
line taller or shorter (`shift+pgup` and `shift+pgdn` five), and `0` lets
it be as tall as its content again. The sizes and the order are kept in
[sticky.json](configuration.md#stickyjson-how-the-board-is-arranged).

## Mouse

| Do this | And |
| --- | --- |
| click a title in the bar | a closed note opens, an open one takes the focus, the focused one closes |
| click a note | it takes the focus |
| click an item, option, button or card | it is ticked, chosen, pressed or selected |
| double-click a note | zoom; on a script, run it after a yes |
| click Yes or No on a question | answer it; a click anywhere else is no |
| wheel | scroll the note under the pointer |
| drag a note's right edge | its width, snapped to the nearest column (the nearest share on `columns auto`) |
| drag a note's bottom edge | its height in lines |
| double-click a bottom edge | as tall as its content again |

In tmux the mouse reaches stickypane only with `set -g mouse on`.

## Editing a note

`e` opens the focused note's file in a small editor that works like vi, so
you can fix a note without leaving the board. `E` hands the file to your own
editor instead: `$VISUAL` or `$EDITOR`, any of them. Terminal editors (vim,
nvim, helix, nano, emacs -nw, micro) take the terminal until you quit; GUI
editors (VS Code, Cursor, Zed, Sublime, JetBrains, Kate, TextMate) are told
to wait until you close the file, so you need not add `--wait`. With neither
variable set, the first of nvim, vim, vi and nano found is used.

| Keys | Do |
| --- | --- |
| `i` `a` `I` `A` `o` `O` | insert text; `esc` goes back |
| `h` `j` `k` `l` `w` `b` `e` `0` `^` `$` `gg` `G` | move |
| `ctrl+f` `ctrl+b`, `ctrl+d` `ctrl+u` | a page, half a page |
| `x` `dd` `dw` `D` `cc` `cw` `C` `r` `J` | delete, change, replace, join |
| `yy` `p` `P`, `u` `ctrl+r` | copy and paste, undo and redo |
| `/text` `n` `N`, `:12` | search, go to a line |
| `:w` `:q` `:wq` `ZZ` `:q!` | save, quit, both, quit without saving |

The border shows the cursor's line and column and which page of the file it
is on. If the file changes on disk while it is open, usually because your
agent wrote to it, a buffer you have not touched follows the file. One with
your changes is kept, you are told, and `:w` does not overwrite the file:
`:w!` does, and `:e!` loads it again. Counts (`3dd`) and visual mode are not
there.

## The index

A board with many notes stays easy to read: `i` or `/` opens the index,
every note of every tab in a line, the way a wiki's index page lists its
pages. Each line has the note's shape, title, what it counts (`2/5`,
`7 cards`), a gist and when it changed; open notes are marked `●`. Type to
filter, choose with the arrows, and `enter` goes to the note, switching tab
and opening it.

![The index: every note of every tab in a line, with what it counts and a gist](../demo/screenshots/4-index.png)

The gist is the note's own `summary:` from its front matter when it has
one, so a note can say in a line what it is; otherwise it is what to look
at first: the next open item of a checklist, the latest line of a log, the
first line of anything else.

`stickypane index` prints the same index as Markdown (`--json` for
programs), and the MCP tool `index` gives it to agents, which read it
instead of every note.

## Deleting and moving

Nothing you do on the board erases a file.

- `dd` asks, then moves the note to `.sticky/.trash/`. `x` moves it to
  `.sticky/archive/` without asking. `u` takes back whichever you did last.
- `m` lists the folders: pick one and the note becomes a page of that book,
  pick the top level to take a page out of its book, or name a new folder.
- `H` and `L` swap the note with its neighbor; the order is kept in
  `sticky.json`.
- `R` renames what you see, not the file: a Markdown note's `title`, or for
  a book, a log or a script the `title` in `sticky.json`.

From the command line:

```sh
stickypane mv plan deploy/       # into the tab deploy
stickypane mv deploy/plan .      # back to the first tab
stickypane mv plan roadmap       # a new name; it keeps its place on the screen
stickypane rm roadmap            # to .sticky/.trash/
stickypane restore roadmap       # and back
stickypane archive roadmap       # to .sticky/archive/
stickypane link README.md        # show a project file without copying it
```

None of these writes over another note or reaches outside `.sticky/`.

## Notes that react

Everything that happens on the board is an event: a note made or removed,
an item ticked, a card moved, a form sent, a line logged, a message said, a
chart value changed. `stickypane watch` prints them as they happen, one a
line, and runs a command for each with `--exec`, so one note can answer
another and anything outside can listen:

```sh
stickypane watch                                   # 14:02:05  plan.md  item.ticked  write tests
stickypane watch --json | jq .                     # for programs
stickypane watch --note plan --type item.ticked \
  --exec 'stickypane chart progress add done 1'    # a chart that counts the ticks
stickypane watch --type card.moved \
  --exec '[ "$STICKY_TO" = Done ] && printf "\a"'  # a bell when a card is done
stickypane watch --once --note deploy --type form.submitted   # wait for the user
```

`--note` and `--type` filter (`--type card` takes every card event), and
`--once` ends at the first event, which is how an agent waits for the user
to do something. The command given to `--exec` runs in your own shell with
the event in `STICKY_EVENT` (JSON) and in `STICKY_NOTE`, `STICKY_KIND`,
`STICKY_TYPE`, `STICKY_ITEM`, `STICKY_FROM`, `STICKY_TO` and `STICKY_TIME`.
Nothing reacts unless you start a watch: the board itself never runs
anything without asking. Over MCP the same events come from `wait_event`.

## How edits are written

The board never overwrites a file with what it last saw. An edit is an
intent ("move this card to that column") applied to the file as it is at
that moment, and the file is then replaced in one step. If your agent
changed that card in the meantime, nothing is written and the board tells
you. A note that is a symlink is edited where it really lives.

Replacing the file has two limits:

- A program that keeps a note open and keeps appending to it
  (`some-command >> .sticky/log.md`) goes on writing to the old file after
  you edit that note from the board. Writing a line at a time, the way agents
  do, is fine.
- A write that lands in the instant between the board reading a file and
  replacing it is lost.

In the code, the intent is a `doc.Op` and `store.Apply` applies it; see
[architecture](architecture.md#reading-and-writing).
