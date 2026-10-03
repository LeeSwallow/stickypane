# Contributing

stickypane has one maintainer and is at the design stage: I am still
deciding how the board should behave, and I change my mind when I see it
used. That shapes what helps most.

## What helps most right now

1. **Use it with your agent for a day and say what happened.** Open a
   [design feedback issue](https://github.com/LeeSwallow/stickypane/issues/new?template=design_feedback.yml):
   what you were trying to do, what you expected the board to show, what it
   did. A text mockup of what you wanted is the best thing you can send.
2. **Vote and argue in [Discussions](https://github.com/LeeSwallow/stickypane/discussions).**
   Ideas is where new things begin; the open design questions are pinned
   there.
3. **Bugs** with the version, the terminal and the pasted screen: the
   [bug template](https://github.com/LeeSwallow/stickypane/issues/new?template=bug.yml).
4. Translations, once the strings are ready for it (see the roadmap).

## Code

Fixes are welcome as pull requests. For anything that changes behaviour,
open a discussion or an issue first: the design is in flux, and I would
rather talk for ten minutes than decline a week of work.

- Go, standard library first. The widget contract is `internal/widget`;
  every shape of note is one package under it and one line in
  `internal/kinds`.
- Tests come with the change: `go vet ./... && go test -race ./...`. The
  test suite is the specification; when a test and the README disagree, say
  so in the PR.
- Write the PR title as a release-note line. Label it `feature`,
  `enhancement`, `bug`, `docs` or `plugin`; the release notes are built
  from the labels.
- The design notes are in `docs/superpowers/specs/`. A change that
  contradicts one should say why.

If you used an AI agent to write the change, that is fine, and this project
is made for that; just read the diff yourself before opening the PR and say
in the description what you checked.

## Not code

The plugin skills (`skills/`), the agent guide (`internal/initcmd/guide.md`)
and the README are where most of the product is. Wording that makes an
agent do the right thing more often is a contribution.
