# Writing a note

Anything else is a file in `.sticky/`. Front matter gives it a `title`, a
shape and `open: true`. `stickypane help <shape>` shows one in full.

- Any Markdown is a note; a ```mermaid block is drawn as a diagram.
- `type: board`: each `## Heading` a column, each `- item` a card.
- `type: checklist`: `- [ ]` and `- [x]` lines. `type: log`, or `name.log`:
  a line per entry. `type: chat`: `@name HH:MM text` lines.
- `type: chart`: `label: number` lines; `view: spark` draws a trend,
  `view: heat` a calendar of dates.
- `type: form`, to ask the user: `- ( ) option` (pick one), `- [ ] option`
  (pick any), an indented line under an option to explain it, `> ` (a line
  to type), and a last line of buttons named by what they do:
  `[ Deploy ] [ Cancel ]`. `stickypane wait <name> --timeout 10m` prints
  the answers. To ask again, leave the `submitted` keys out.
- `name.sh`: a script the user runs after a yes. A folder: a tab; a folder
  in a tab: a book.

Write in the user's language. A title is what the user would call it; an
item or a card is one action, verb first; a log line is one event, with
the number that matters. One note per thing the user follows: update it,
do not add a second. Show what to read now and `hide` the rest. Point to a
file with `stickypane show`; do not paste it.
