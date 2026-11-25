# Changing the board

Each command changes one thing and prints where the note stands
(`plan.md: 2/5`): report that line, do not read the file back. A missing
note is made, open, and so is a missing board in a git repository
(elsewhere `stickypane init`).

```sh
stickypane note show README.md               # a note, or any file or folder of the project
stickypane todo add plan "write tests"       # also: check, uncheck <item>
stickypane card move work login --to Done    # also: add "text" --to Doing
stickypane chart add tokens input 1200       # also: set
stickypane log add worklog --time "tests passed"
stickypane chat add chat "41/41 pass"        # signed "agent"; --as <your name> to sign it
stickypane note set plan open=true size=1/2  # also: title="..."; note hide <name> folds it
stickypane note mv plan deploy/              # into a tab; also: note rm, restore, link <path>
```

An item, a card or a column is named by its text, a part of it, or `#2`.

- Never edit `sticky.json`: it is how the user arranged the screen.
- Do not write times, or the numbers of a `from:` chart: the board does.
- The user edits too: run `stickypane note cat <name>` before rewriting a note.
- In a worktree, write in the folder the session names: `worktrees/<name>/`.
- Read a form with `stickypane wait` or `answers`, a chat with `watch`. Do
  what is hard to undo only after a fresh form, showing its effect, is
  `submitted`; exit code 3 is a no: take the safe path and log it.
- Run nothing (`watch --exec`) and remove nothing unless the user asks.
