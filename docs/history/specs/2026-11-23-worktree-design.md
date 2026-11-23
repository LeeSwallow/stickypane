# Worktrees on the board

Design, 2026-11-23. Status: the design the proposals [#140](https://github.com/LeeSwallow/stickypane/issues/140) to #149 were written from. Where the two differ, the issues win: what the use-case research changed (R1 to R11) is in [worktree-issues/english-draft-and-revisions.md](2026-11-23-worktree-issues/english-draft-and-revisions.md).

## Intent

A coding agent often works in a git worktree, one per branch, and several
at once. stickypane has no idea of a worktree today: `.sticky/` is
gitignored, so a new worktree has no board at all, and the only trace of a
branch is a `@feat/chart-ui` tag typed on a kanban card
(`.sticky-backup-20261003-1846/05-parallel.md`).

The user wants the board to handle worktrees properly: remember where each
one is, and keep a page (a folder of notes) for each, so the work of every
worktree can be told apart on one board.

## Goals

- One board for a repository, shared by all its worktrees. An agent that
  starts in a worktree finds the main repository's `.sticky/`.
- Each worktree has its own folder on that board, made without anyone
  asking, where its agent keeps that worktree's checklist and notes.
- The board knows where each worktree is, and notices when one is moved or
  removed.
- The overall board can count what the worktree folders hold, with the
  `from:` charts it already has.
- A worktree that is gone leaves its folder behind, marked closed.

## Non-goals

- Making, moving or deleting worktrees. Orca, Claude Code and `git` do
  that; the board follows.
- The diff view (changed files and the code of each). It is a separate
  feature, designed after this one, and uses a worktree's path from here.
- Syncing a board between computers.

## Anti-goals

- A second source of truth for where a worktree is. If the board and git
  disagree, git is right.
- A board folder that stops working when it is committed to a repository
  and read on another computer.
- Anything the agent must learn: the guide stays within its 55 lines.

## Constraints

From VISION.md and AGENTS.md: the folder is the product, and the board
keeps no state that is not a file in it, except what belongs to one
computer; standard library first; every change to a note is a `doc.Op`
through `store.Apply`; times are written by the board through
`internal/core/when`; the layers in `internal/layers`.

## Design

### Which board a worktree uses

`git worktree list --porcelain` names the main worktree first. The board of
a repository is the `.sticky/` of its main worktree. A process started
inside any other worktree uses that one, unless the worktree has a
`.sticky/` of its own, which wins (a person who wants a separate board
keeps getting it).

### The folder of a worktree

```
.sticky/
  worktrees/
    feat-chart-ui/
      00-개요.md      what the branch is for, its state; for people
      01-할일.md      a checklist for this worktree
```

The folder name is the branch with `/` changed to `-`; a worktree with no
branch (detached) is named by the last element of its path. The first note
is made when a board opens in a worktree that has no folder, and by the
rest of the board's commands too (`stickypane` in that worktree). Nothing
to run by hand.

`00-개요.md` carries only what a person reads: the branch, what it is for,
a state line. It does not carry the path.

### Where a worktree is

Two files in `~/.stickypane` (or `$STICKYPANE_HOME`), kept by one writer in
`internal/adapter/registry`:

- `worktrees.json`, the state now, a JSON object with a `version` and, per
  board, a list of `{name, path, branch, state}`. `state` is `open` or
  `closed`. A computer's paths stay on that computer: a committed
  `.sticky/` never carries one.
- `events/<date>.jsonl`, the history: `worktree.open`, `worktree.close`,
  `worktree.move`, one line each, with the name and the paths.

Format is JSON because everything else in the home is, and `encoding/json`
needs nothing new. Like the rest of the registry, writing is best-effort and
only happens where the desktop app has written its `overlay.json`; without
it the board still works: it reads `git worktree list` each time and the
folders carry the rest.

### Keeping the two in step

When a board opens (and when a worktree's agent runs a command), the board
reads `git worktree list --porcelain` and, for each worktree:

| Git says | Board does |
| --- | --- |
| a worktree it has no folder for | makes the folder and its first notes; records it `open` |
| the branch is the same, the path changed | updates the path; records `worktree.move` |
| the worktree is gone or `prunable` | marks it `closed` in the registry and in `00-개요.md`; keeps the folder |
| a folder with no worktree and no record | leaves it alone (a person made it) |

A worktree is matched by its branch; a detached one by its path. If git is
not installed or the folder is not a repository, none of this runs and the
board is exactly what it was.

### What the agent sees

A line in the guide's rules: in a worktree, keep notes in
`worktrees/<name>/`; the board says the name. The overall board's
`진행.md` counts the worktree folders with `from:`.

## Approaches

1. **One shared board, the worktree folder inside it. Chosen.** One screen
   to watch; survives the worktree being deleted; the folder gives each
   worktree its own place. Cost: a shared folder gets busier.
2. **A board in every worktree.** Isolated, but `.sticky/` is gitignored, so
   the notes die with the worktree, and the user watches several boards.
3. **A registry only, no folders.** Cheapest, but gives no page per
   worktree, which is what was asked.

Formats for where a worktree is: JSON (chosen); YAML or TOML (a new module
each, and written by the board, not by hand); SQLite (a binary file, a
module, and a dependency for a handful of rows); a field in the note's
front matter (a path is a fact of one computer and git already knows it, so
it does not belong to a document).

## Left to the builder

- The exact words of the guide line and of the notes (Korean and English,
  in the voice of the other notes).
- Whether the folder is made by `store` or by a use case; the layers decide.
- How the paths are shown in the desktop app.
- Tests: a fake `git worktree list` output in `testdata`, golden like the
  registry's home.

## My calls, to check

- The folder name rule (`/` to `-`, a detached worktree by its path). Two
  worktrees that would get the same name get a short hash of the path
  after it (`feat-a-b-3f9c`). A branch that comes back reopens its closed
  folder and sets the state to `open`. A renamed branch reads as gone plus
  new; the old folder stays closed. Accepted.
- Every command resolves `.sticky/` the same way (the worktree's own if
  that folder exists at its root, else the main worktree's), not only the
  TUI and the app. The root-finding code is one place, so this is one
  change.
- The main worktree gets no folder: its board is the board.
- A worktree's own `.sticky/` wins over the main one.
- A closed worktree's folder is kept, never deleted by the board.
- The first note is named `00-개요.md`. Its state is front matter
  (`state: open|closed`, and `closed: <time>` written through `when`), set
  by a `doc.Op` through `store.Apply`; a person may edit the rest freely.
- The registry is keyed by the main worktree's `git rev-parse
  --git-common-dir` (absolute). With the registry off, nothing is lost:
  paths are read from git every time, and `closed` lives in `00-개요.md`.
- Closed means git itself says `prunable` or the worktree is not listed.
  A path that is only unmounted for a while is still listed, so it stays
  open. A `locked` worktree stays open.
- Races: the folder is made under the existing lock on the notes folder
  (`filelock`), and the registry is written with `atomicfile`; two agents
  making the same folder end with one.
- `진행.md` is not made by this feature; a board that has one counts the
  worktree folders with `from:` (whether `from:` reaches into nested
  folders is checked in the plan).

## Open: for the person to decide

How an agent learns its worktree's folder name. Proposed: the session hook
that already prints the board prints one more line in a worktree
(`worktree: feat-chart-ui → worktrees/feat-chart-ui/`), and commands do
not change. The agent writes its own notes there by path. The other way,
commands defaulting new notes into the worktree's folder, is a larger
change to every command and puts files where the agent did not name them.
