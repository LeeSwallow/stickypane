# The shapes of a note

One entry per shape: what it is for, the fastest way to make one, the file,
and how it behaves on the board. `stickypane kinds` lists the shapes and
`stickypane kinds <shape>` prints the full example of one.

## Note

For anything to read: an explanation, a decision, a summary.
`echo "check env before deploy" | stickypane write deploy-notes --open`

```markdown
---
title: Why the cache
---
Any Markdown. A ```mermaid block (graph, flowchart, sequence, erDiagram) is drawn.
```

A one-line file is a complete note. A diagram that cannot be drawn keeps
its source.

## Board

For work in columns. `stickypane card work add "login API" --to Doing`

```markdown
---
type: board
---
## To do
- payments
## Doing
- login API
  - refresh tokens come later
```

Each `## Heading` is a column and each top-level `- item` a card; indented
lines are the card's details. The user moves cards with keys and the file
changes. `stickypane card work move login --to Done` moves one.

## Checklist

For a list of steps. `stickypane todo plan add "write tests"`

```markdown
---
type: checklist
---
- [x] add endpoint
- [ ] write tests
```

Shown with a progress bar. The user ticks items with `space`;
`stickypane todo plan check tests` ticks one from your side.

## Log

For what happened, in order. `stickypane log worklog --time "tests passed"`

A note with `type: log`, one line per entry, or any `.log`, `.txt` or
`.out` file. The view follows the end as the file grows until the user
scrolls up. `stickypane show logs/app.log` follows a log of the project.

## Chart

For numbers. `stickypane chart tokens add input 1200`

```markdown
---
type: chart
view: heat
---
2026-10-01: 41,200
2026-10-02: 8,900
```

Each `label: number` line is a value; other lines are text above it.
`view: bar` (default) draws bars, `spark` a one-line trend, `heat` a
calendar when the labels are dates.

## Form

For a decision that is the user's. Write it, show it, wait:
`stickypane wait deploy --timeout 10m`

```markdown
---
type: form
title: Deploy now?
---
## Target
- ( ) staging
- ( ) production

[ Deploy ] [ Cancel ]
```

`- ( )` is one choice, `- [ ]` any number, `> ` a line to type into, and
the last line the buttons. A press writes `submitted:` into the front
matter and `wait` prints the answers. See the `asking-the-user` skill.

## Script

For a command the user should run, not you: `deploy.sh` in `.sticky/`, or
`deploy.ps1` for PowerShell. `enter` asks first, then runs it with `sh` or
PowerShell in the project folder; the output
goes to `deploy.log`, which opens and follows. It never runs without a yes.

## Tab

For a separate screen: a folder of `.sticky/`, such as `deploy/`. Its
files are the tab's notes. `stickypane mv plan deploy/` moves a note there;
`stickypane show deploy/plan` switches to the tab and shows it. Linking a
folder of the project (`stickypane show docs/`) makes it a tab.

## Book

For pages read in order: a folder inside a tab, such as `deploy/guide/`.
It is one note whose pages are its files, by name; `,` and `.` turn pages.
