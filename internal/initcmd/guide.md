<!-- stickypane:start -->
## stickypane notes

The user keeps a note board open in a terminal pane next to you. It shows the
Markdown files in `.stickypane/`: one file is one note. Create a file to stick a
note, edit it to update the note, delete it to take the note down.

Jot freely: decisions, things to remember, progress. A one-line file is a
complete note. Add front matter only when a note needs a shape:

- Plain note: any Markdown, from one line to a full page. A ```mermaid block
  (flowchart, sequence or ER diagram) is drawn as a diagram.
- Any note takes these optional keys: `title`; `open: true` to put it on the
  screen now (`open: false` to keep it folded away); `size` (`page` for the
  whole width, `half`, or `card`); `pin: true`; `color` (yellow, pink, blue,
  green, purple, orange).
- Board: `type: board`. Each `## Heading` is a column, each top-level `- item`
  below it is a card, and indented lines under a card are its details. Move a
  card by moving its lines under another heading.
- Checklist: `type: checklist`. Items are `- [ ]` and `- [x]` lines.
- Log: `type: log`. Append one line per entry at the end of the file.
- Chart: `type: chart`. Each `label: number` line is a bar. `view: spark` draws
  a trend line; `view: heat` a calendar when labels are dates (`2026-10-01: 4`).
- Form: `type: form`, to ask the user. Markdown with `- ( ) option` lines
  (choose one), `- [ ] option` lines (choose any), a `> ` line (text to fill
  in) and a line of buttons. The answers are written into the file: run
  `stickypane wait <name>` to block until a button is pressed and print them.
  A pressed button adds `submitted` keys; leave them out when you ask again.

    ---
    type: form
    title: Deploy now?
    ---
    - ( ) staging
    - ( ) production
    [ Deploy ] [ Cancel ]

The screen shows open notes in full and lists the others by title, so open
what the user should read now. When progress changes, update the matching
note instead of adding a new one, and keep the keys you are not changing:
`open` and `size` are how the user arranged the screen. The user can edit
notes from the board, so read a note again before you change it. Notes are
ordered by file name; prefix a number (`10-plan.md`) to control the order.
<!-- stickypane:end -->
