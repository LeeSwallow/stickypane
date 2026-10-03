# File formats

This is the guide `stickypane init` installs, word for word; `stickypane
guide` prints the current one. Markdown gets its shape from front matter;
other files from their extension.

## stickypane notes

The user keeps a board open in a terminal pane next to you. It shows the files
in `.sticky/`: one file is one note. Create a file to stick a note, edit it to
update the note. Jot freely: decisions, things to remember, progress. A
one-line file is a complete note.

What a file is:

- `name.md`: a note. Any Markdown; a ```mermaid block (flowchart, sequence or
  ER diagram) is drawn. Front matter gives it a `title` and a shape:
  - `type: board`: each `## Heading` is a column, each top-level `- item`
    under it a card, indented lines under a card its details.
  - `type: checklist`: `- [ ]` and `- [x]` lines.
  - `type: log`: one line per entry, appended at the end.
  - `type: chart`: `label: number` lines as bars. `view: spark` draws a trend,
    `view: heat` a calendar when labels are dates (`2026-10-01: 4`).
  - `type: form`, to ask the user: `- ( ) option` (choose one), `- [ ] option`
    (choose any), a `> ` line (text to fill in), a line of buttons such as
    `[ Deploy ] [ Cancel ]`. Answers are written into the file;
    `stickypane wait <name>` blocks until a button is pressed and prints
    them. Leave the `submitted` keys out when you ask again.
- `name.log` (or `.txt`, `.out`): a log, shown as it is and followed live.
- `name.sh`: a script the user runs with a button; its output goes to
  `name.log`. It never runs without the user saying yes.
- A folder: one note with pages, a page per file (`docs/intro.md`).

To put something in front of the user, `stickypane show <name>`; a path to
any file or folder of the project (`README.md`, `logs/app.log`, `docs/`) is
linked onto the board and shown, a log followed live. `hide` folds it away.

For a small change do not read or rewrite the note: run one of these. They
make the note when it is missing and print where it stands (`plan.md: 2/5`).
An item or a card is named by its text, a part of it, or its position (`#2`).

    stickypane todo plan add "write tests"       # also: check, uncheck
    stickypane card work move "login" --to Done  # also: add "text" --to Doing
    stickypane chart tokens add input 1200       # also: set input 5000
    stickypane log worklog --time "tests passed"
    stickypane set plan open=true size=half      # show it; page, half or card
    stickypane mv plan docs/                     # also: rm, restore, link <path>

The screen shows open notes side by side, each in a pane, and lists the others
by title, so show what the user should read now (`stickypane show`, or
`open: true` in a new note's front matter). Where notes are on the screen is the user's
to arrange and is kept in `sticky.json`; do not edit that file. When progress
changes, update the matching note instead of adding a new one. The user can
edit notes from the board, so read a note again before you rewrite it. Notes
are ordered by file name; prefix a number (`10-plan.md`). Write notes in the
language the user writes in.
