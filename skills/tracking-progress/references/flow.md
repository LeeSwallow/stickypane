# Flow

## Start

A checklist for a list of steps; a kanban when the work has columns.

```sh
stickypane todo refactor add "find every caller"      # makes refactor.md, open
stickypane todo refactor add "move the function"
stickypane todo refactor add "run the tests"
# or
stickypane card work add "login API" --to "To do"
```

## While working

```sh
stickypane todo refactor check "callers"              # a part of the text, or #1
stickypane log worklog --time "tests passed, 3 flaky ones left"
stickypane chart tokens add input 1200
stickypane card work move "login" --to Done
```

Each command prints where the note stands (`refactor.md: 2/3`). Report
that; do not read the note.

## Finish

Check the last item and write one log line with the outcome:

```sh
stickypane todo refactor check "tests"
stickypane log worklog --time "refactor done: 14 files, tests green"
```
