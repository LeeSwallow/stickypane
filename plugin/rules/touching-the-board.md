# Changing the board

Each command changes one thing and prints where the note stands
(`plan.md: 2/5`): report that line, do not read the file back. A missing
note is made, open, and so is a missing board in a git repository
(elsewhere `stickypane init`).

```sh
stickypane show README.md                  # a note, or any file or folder of the project
stickypane todo plan add "write tests"     # also: check, uncheck <item>
stickypane card work move login --to Done  # also: add "text" --to Doing
stickypane chart tokens add input 1200     # also: set
stickypane log worklog --time "tests passed"
stickypane say chat "41/41 pass"           # signed "agent"; --as <your name> to sign it
stickypane set plan open=true size=1/2     # also: title="..."; hide <name> folds it
stickypane mv plan deploy/                 # into a tab; also: rm, restore, link <path>
```

An item, a card or a column is named by its text, a part of it, or `#2`.

- Never edit `sticky.json`: it is how the user arranged the screen.
- Do not write times, or the numbers of a `from:` chart: the board does.
- The user edits from the board too. Use the commands, and run
  `stickypane cat <name>` before you rewrite a note.
- Read a form with `stickypane wait` or `answers`, a chat with `watch`.
- Run nothing (`watch --exec`) and remove nothing unless the user asks.
