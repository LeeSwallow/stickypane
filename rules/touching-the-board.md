# What the board keeps for itself

Every skill of this plugin follows these. They are what keeps an agent and
the user from undoing each other.

- **The arrangement is the user's.** Never edit `sticky.json`; change how a
  note is shown with `stickypane set`, `show` or `hide`, and only when the
  user would expect it.
- **Times are the board's.** Do not write when something happened: the
  board writes `created:`, the time an item is ticked (`✅ 2026-10-03
  14:02`), a card's `@{date}`, a message's time and a log line's `--time`.
- **Computed values are the board's.** A chart with `from:` is filled in by
  the board; do not write its numbers.
- **Change, do not rewrite.** A one-line command changes one thing in the
  file as it is now. Rewriting a note loses what the user ticked, moved or
  typed since you read it; read it first when you must.
- **Nothing runs without a yes.** Scripts run when the user presses Run and
  confirms. `stickypane watch --exec` runs commands in the user's shell:
  start one only when the user asked for that reaction.
- **Deleting is the user's call.** `stickypane rm` moves to `.trash/` and
  `restore` brings it back; still, remove a note only when asked.
- **The answer comes from the board.** Read a form with `wait` or
  `answers` and a chat with `watch`, not by parsing the marks yourself.
