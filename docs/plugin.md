# The plugin

This repository is also a Claude Code and Codex plugin. For how to use the
board, see the [README](../README.md); this page is for changing the plugin.

## Layout

| Path | Is |
| --- | --- |
| `.claude-plugin/plugin.json`, `marketplace.json` | the Claude Code plugin `board` in the marketplace `stickypane` |
| `.codex-plugin/plugin.json`, `.agents/plugins/marketplace.json` | the same for Codex |
| `skills/<name>/SKILL.md` | a skill: its name, when to use it (the description), and loading instructions |
| `skills/<name>/references/` | what the skill loads: `rules.md` (boundaries), `flow.md` (the procedure), and shapes to copy |
| `rules/` | rules every skill shares |
| `commands/<name>.md` | the slash commands `/board:<name>` |
| `hooks/` | hook registrations for each agent; both run the scripts below |
| `scripts/` | shell that the hooks and commands run |

A skill's `SKILL.md` stays short: it says when the skill applies and which
reference to read for what. Values that live elsewhere are not copied into
the skills: the file formats come from `stickypane guide`, and
`references/formats.md` is checked against it by `go test` and
`scripts/selfcheck.sh`.

## Scripts

| Script | Usage | Does |
| --- | --- | --- |
| `scripts/board-context.sh` | no arguments, in a project | prints `STICKYPANE_INSTALLED`, `STICKYPANE_VERSION`, `BOARD_DIR` and the note list |
| `scripts/session-start.sh` | run by the SessionStart hook | tells the agent what is on the board; silent without one |
| `scripts/selfcheck.sh` | no arguments | checks manifests, scripts, skills, commands and the formats reference |

Exit codes: `0` fine; `1` selfcheck found a problem (listed with ❌); `3`
no board at or above the current directory; `4` stickypane is not
installed.

## Adding a skill

1. Make `skills/<name>/SKILL.md` with `name`, a `description` that starts
   with "Use when", and loading instructions.
2. Put the boundaries in `references/rules.md` and the steps in
   `references/flow.md`.
3. Run `scripts/selfcheck.sh` and `go test ./internal/initcmd/`.
