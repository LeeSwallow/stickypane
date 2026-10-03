# Flow

## 1. Something already exists: show it

```sh
stickypane show README.md        # any file of the project, linked onto the board
stickypane show logs/app.log     # a log: shown and followed as it grows
stickypane show docs/            # a folder: one note with a page per file
stickypane show plan             # a note that is already on the board
stickypane hide plan
```

## 2. One thing changes: run a command

The note is made when it is missing. An item, a card or a column is named
by its text, a part of it that nothing else has, or its position (`#2`).
When a name fits nothing or more than one thing, the error lists what there
is; pick from that list instead of reading the file.

| Command | Does | Prints |
| --- | --- | --- |
| `stickypane todo <note> add "text"` | adds a checklist item | `plan.md: 0/3` |
| `stickypane todo <note> check\|uncheck <item>` | ticks it | `plan.md: 1/3` |
| `stickypane card <note> add "text" [--to <column>]` | adds a kanban card | `work.md: 4 cards` |
| `stickypane card <note> move <card> --to <column>` | moves it | `work.md: 4 cards` |
| `stickypane chart <note> set <label> <number>` | sets a value | `tokens.md: input = 5000` |
| `stickypane chart <note> add <label> <number>` | counts it up or down | `tokens.md: input = 6200` |
| `stickypane log <note> [--time] "text"` | appends a line | `worklog.md: 12 lines` |
| `stickypane set <note> key=value ...` | front matter (`title`, `type`, `view`) or arrangement (`open`, `size`, `rows`, `color`, `pin`) | `plan.md: set title` |
| `stickypane mv <note> <docs/\|.\|new-name>` | moves or renames | `moved plan.md to docs/plan.md` |
| `stickypane rm <note>` / `restore <note>` | to the trash and back | `moved … to .trash/…` |

## 3. A shape of its own: write a file

Write the Markdown file into the notes folder, then `stickypane show` it.
The formats are in `references/formats.md`. Front matter `open: true`
shows it at once without the extra command.

## 4. Reading the board

`stickypane list` for what exists and what is open; `stickypane cat <name>`
for one note's file; `stickypane answers <form>` for what the user chose.
Do not read every note to answer "what is on the board": the list is the
answer.
