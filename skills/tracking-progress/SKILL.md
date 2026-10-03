---
name: tracking-progress
description: Keep the progress of a multi-step task visible on the user's stickypane board while you work: a checklist or kanban for the steps, a log for what happened, a chart for counts. Use when a task has more than a few steps, when it will take long enough that the user may look away, and whenever the user asks "where are we".
---

# Tracking progress on the board

The user should be able to glance at the board and know where the work
stands, without reading the chat. Keep one note per task and update it as
you go.

## Start

```sh
stickypane todo refactor add "find every caller"      # makes refactor.md
stickypane todo refactor add "move the function"
stickypane todo refactor add "run the tests"
```

A kanban fits work with columns (`stickypane card work add "login API"
--to Doing`); a checklist fits a list of steps. Pick one.

## While working

```sh
stickypane todo refactor check "callers"              # by a part of the text, or #1
stickypane log worklog --time "tests passed, 3 flaky ones left"
stickypane chart tokens add input 1200                # counts go up
stickypane card work move "login" --to Done
```

Each command prints where the note stands (`refactor.md: 2/3`), so you do
not read the note to know.

## When something needs the user

Ask on the board (the `asking-the-user` skill) instead of leaving a question
in the chat where it scrolls away.

## Finish

Check the last item, write one log line with the outcome, and leave the
note where it is: it is the record. Do not delete it unless the user asks.

## Rules

- Update the note that exists. A second checklist for the same task is
  noise.
- One line per log entry, with `--time` when the order of events matters.
- Never rewrite a checklist to tick an item: the user may have ticked or
  added items from the board. The one-line commands keep their changes.
