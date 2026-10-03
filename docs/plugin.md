# The plugin

This repository is also a Claude Code and Codex plugin. For how to use the
board, see the [README](../README.md); this page is for changing the plugin.

## Layout

| Path | Is |
| --- | --- |
| `.claude-plugin/plugin.json`, `marketplace.json` | the Claude Code plugin `board` in the marketplace `stickypane` |
| `.codex-plugin/plugin.json`, `.agents/plugins/marketplace.json` | the same for Codex |
| `skills/<name>/SKILL.md` | a skill: its name, when to use it (the description), then the commands and the rules |
| `skills/using-the-board/references/kinds.md` | one short entry per shape of note, read when a shape is needed |
| `rules/` | what every skill keeps to: what the board writes itself, how a note is worded |
| `commands/<name>.md` | the slash commands `/board:<name>`: a line that names the skill to follow |
| `harness.go`, `internal/harness` | the same skills carried in the program, for MCP prompts and `stickypane guide` |
| `hooks/` | hook registrations for each agent; both run the scripts below |
| `scripts/` | shell that the hooks and commands run |

The plugin is layered: a command names a skill, a skill points to its
references and to the shared rules, and `using-the-board` is the way in
that names every other skill. A skill opens with what to run, so an agent
can act after reading its first screen; the rules follow. Each thing is said in one place: the shapes in
`references/kinds.md`, forms in `asking-the-user`, progress in
`tracking-progress`. The guide `stickypane init` writes into `AGENTS.md`
(`internal/initcmd/guide.md`) is the same knowledge for agents without the
plugin, kept to one screen. `go test ./internal/initcmd/` checks that the
skills stay short, name only files that exist and run only real commands.

## Scripts

| Script | Usage | Does |
| --- | --- | --- |
| `scripts/board-context.sh` | no arguments, in a project | prints `STICKYPANE_INSTALLED`, `STICKYPANE_VERSION`, `BOARD_DIR` and the note list |
| `scripts/session-start.sh` | run by the SessionStart hook | tells the agent what is on the board; silent without one |
| `scripts/selfcheck.sh` | no arguments | checks manifests, scripts, skills and commands |

Exit codes: `0` fine; `1` selfcheck found a problem (listed with ❌); `3`
no board at or above the current directory; `4` stickypane is not
installed.

## Adding a skill

1. Make `skills/<name>/SKILL.md` with `name` and a `description` that
   starts with "Use when". Put the commands first and the rules after, in
   under 80 lines.
2. Run `scripts/selfcheck.sh` and `go test ./internal/initcmd/`.
