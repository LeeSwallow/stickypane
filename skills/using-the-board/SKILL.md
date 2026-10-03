---
name: using-the-board
description: Use when the user asks to show, track, list, chart, draw or explain something "on the board", when a plan, a checklist, a log or a diagram is better seen than read in chat, and before saying what the board shows. Puts files on the user's stickypane board (a terminal pane next to you) and changes them with one-line commands.
---

# Using the board

The board is the project's `.sticky/` folder, shown by `stickypane` in a
pane next to you. One file is one note. Every command below prints one line
saying where the note stands; report it and do not read the file back. A
missing note is made, and so is a missing board in a git repository.

## 1. It exists already: show it

```sh
stickypane show README.md        # any file of the project, linked onto the board
stickypane show logs/app.log     # a log, followed as it grows
stickypane show plan             # a note that is on the board
stickypane hide plan             # fold it away
```

## 2. One thing changes: run a command

```sh
stickypane todo plan add "write tests"         # plan.md: 0/3   (check, uncheck)
stickypane card work move "login" --to Done    # work.md: 4 cards   (add "text" --to Doing)
stickypane chart tokens add input 1200         # tokens.md: input = 6200   (set)
stickypane log worklog --time "tests passed"   # worklog.md: 12 lines
stickypane set plan title="The plan" open=true size=half
stickypane mv plan deploy/                     # into the tab deploy (rm, restore)
```

An item, a card or a column is named by its text, a part of it that
nothing else has, or its position (`#2`). When a name fits nothing or more
than one thing, the error lists what there is: pick from it instead of
reading the file.

## 3. A shape of its own: write a file

Write the Markdown file into `.sticky/` with `open: true` in its front
matter, or write it and run `stickypane show <name>`.

| Shape | Front matter | Body |
| --- | --- | --- |
| note | none | any Markdown; a `mermaid` block is drawn |
| board | `type: board` | `## Column`, then `- card` lines |
| checklist | `type: checklist` | `- [ ]` and `- [x]` lines |
| log | `type: log`, or a `.log` file | one line per entry |
| chart | `type: chart`, `view: bar\|spark\|heat` | `label: number` lines |
| form | `type: form` | see the `asking-the-user` skill |
| script | a `.sh` file | shell; the user runs it after a yes |

What each shape is for and how it behaves: `references/kinds.md`. A full
example of one: `stickypane kinds <shape>`.

## 4. Reading the board

`stickypane list` says what exists and what is open; that is the answer to
"what is on the board". `stickypane cat <name>` prints one note;
`stickypane answers <form>` what the user chose.

## Rules

- Prefer, in this order: `show`, a one-line command, writing a file. Each
  step down costs more reading and more risk of overwriting the user.
- The user edits notes from the board. Run `stickypane cat <name>` before
  rewriting a note; the one-line commands never overwrite.
- One note per thing the user follows. Update it; do not add a second.
- Show what the user should read now; leave the rest folded.
- Never edit `sticky.json`: it is how the user arranged the screen.
- Names: `plan` or `plan.md`; a note in a tab or a book is `deploy/plan`;
  a log is `build.log`.
- Write in the user's language. A title is a noun phrase ("Auth refactor").
  An item or a card is one action, verb first, under ten words. A log line
  is one event, past tense, with the number that matters ("tests passed, 3
  flaky left"). Use a list, a diagram or a chart where prose would be long.
