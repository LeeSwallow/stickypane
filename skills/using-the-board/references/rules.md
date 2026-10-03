# Rules

- The board is the project's `.sticky/` folder (or `.stickypane/`), shown by
  `stickypane` in a pane next to you. One file is one note. The user reads
  the board instead of the chat for anything that should stay in view.
- Prefer, in this order: `stickypane show <file>`; a one-line command
  (`todo`, `card`, `chart`, `log`, `set`); writing a Markdown file. Each
  step down costs more reading and more risk of overwriting the user.
- Never edit `sticky.json`. It is how the user arranged the screen.
- The user changes notes from the board. Before rewriting a note, read it
  (`stickypane cat <name>`); the one-line commands never overwrite, so use
  them for small changes.
- One note per thing the user follows. Update it; do not add a second.
- Show what the user should read now (`stickypane show`). Leave the rest
  folded: the user opens what they want.
- Every command prints one line saying where the note stands. Report that
  line; do not read the file back to confirm it.
- Names: a note is `plan` or `plan.md`; a page of a folder is `docs/plan`;
  a log is `build.log`. `stickypane list` shows what exists.
