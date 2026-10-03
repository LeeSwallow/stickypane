<!-- stickypane:start -->
## stickypane notes

The user keeps a board open in a terminal pane next to you. It shows the
files in `.sticky/`: one file is one note. Put there what the user should
keep in view (a plan, progress, a decision to make) instead of the chat.

Put something on it with one command. Each prints where the note stands
(`plan.md: 2/5`); report that line, do not read the file back. A missing
note is made, and so is a missing board in a git repository.

    stickypane show README.md                    # a note, or any file or folder of the project
    stickypane todo plan add "write tests"       # also: check, uncheck <item>
    stickypane card work move "login" --to Done  # also: add "text" --to Doing
    stickypane chart tokens add input 1200       # also: set input 5000
    stickypane log worklog --time "tests passed"
    stickypane set plan open=true size=half      # page, half or card; hide <name> folds it
    stickypane mv plan deploy/                   # also: rm, restore, link <path>

An item or a card is named by its text, a part of it, or its position (`#2`).

For anything else, write a file into `.sticky/`. Front matter gives a
Markdown note a `title` and a shape; `stickypane kinds <shape>` prints a
full example of one.

- A plain note: any Markdown. A ```mermaid block (flowchart, sequence or
  ER diagram) is drawn.
- `type: board`, a kanban: each `## Heading` is a column, each top-level
  `- item` under it a card, indented lines under a card its details.
- `type: checklist`: `- [ ]` and `- [x]` lines, with a progress bar.
- `type: log`: one line per entry; the view follows the end.
- `type: chart`: `label: number` lines as bars. `view: spark` draws a trend,
  `view: heat` a calendar when the labels are dates (`2026-10-01: 4`).
- `type: form`, to ask the user: `- ( ) option` (choose one), `- [ ] option`
  (choose any), a `> ` line (text to fill in), a line of buttons such as
  `[ Deploy ] [ Cancel ]`. `stickypane wait <name> --timeout 10m` blocks
  until a button is pressed and prints the answers. To ask again, leave the
  `submitted` keys out.
- `name.log` (or `.txt`, `.out`): a log, shown as it is and followed live.
- `name.sh`: a script the user runs with a button, after saying yes. Its
  output goes to `name.log`.
- A folder: a tab, a screen of its own. A folder in a tab: a book, one note
  with a page per file.

Rules:

- Show what the user should read now (`stickypane show`, or `open: true` in
  a new note's front matter). The rest stays folded.
- One note per thing the user follows. Update it; do not add a second.
- The user edits notes from the board. Use the commands for small changes,
  and `stickypane cat <name>` before you rewrite a note.
- Never edit `sticky.json`: it is how the user arranged the screen.
- Prefix a number to order notes (`10-plan.md`). Write in the user's
  language: short titles, one action per item, one event per log line.
<!-- stickypane:end -->
