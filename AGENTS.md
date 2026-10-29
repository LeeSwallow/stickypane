# AGENTS.md

What a coding agent needs before changing this repository. People: the
same is in [CONTRIBUTING.md](CONTRIBUTING.md), at more length.

## What this is

stickypane is a Go program that shows a folder of plain files (`.sticky/`)
as a board in a terminal pane, for a coding agent and its user. The
principles are in [VISION.md](VISION.md); the code's layout in
[docs/architecture.md](docs/architecture.md).

## Before you finish

```sh
go vet ./... && go test -race ./... && test -z "$(gofmt -l .)"
```

All three must pass. The tests are the specification: change a test only
when the behaviour is meant to change, and say so.

## Rules

- **Layers.** A package sits in the folder of its layer, `internal/core`,
  `internal/adapter`, `internal/usecase` or `internal/surface`, and
  imports only packages of a lower rank in its own folder or a lower one;
  `internal/layers` fails otherwise. A new package gets a
  folder, a rank there and a line in docs/architecture.md.
- **A shape of note** is one package under `internal/core/widget/` with
  `kind.go`, `format.go`, `model.go`, `ops.go`, `view.go`, `events.go`, and
  one line in `internal/core/kinds`. Every change to a note is a `doc.Op`
  applied by `store.Apply`; never write a note's file directly.
- **Times** are written by the board through `internal/core/when`, never by an
  agent or by hand in a note.
- **Standard library first.** A new module needs a reason, and a license
  that `sh scripts/licenses.sh` accepts.
- **Words.** The README, the skills and the agent guide
  (`plugin/guide.md` with `plugin/rules/`, at most 55 lines) change
  with the behaviour. Each rule is written once, in `plugin/rules/`. The skills
  are tested: short, opening with a command, naming only real commands and
  files.
- **Commits** are Conventional Commits, one change each. Do not add
  `Co-Authored-By` or "Generated with" lines naming an AI tool; the person
  who commits is the author. Run `git config core.hooksPath .githooks` so
  the hook removes them if your tool adds them.
- **Ask first** before pushing, merging into `main`, tagging, deleting
  branches, or anything that leaves this machine.
