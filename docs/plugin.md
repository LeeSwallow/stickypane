# The plugin

This repository is also a Claude Code and Codex plugin. For how to use the
board, see the [README](../README.md); this page is for changing the plugin.

## Layout

| Path | Is |
| --- | --- |
| `.claude-plugin/plugin.json`, `marketplace.json` | the Claude Code plugin `board` in the marketplace `stickypane` |
| `.codex-plugin/plugin.json`, `.agents/plugins/marketplace.json` | the same for Codex |
| `skills/<name>/SKILL.md` | a skill: its name, when to use it (the description), then the commands and the rules |
| `skills/using-the-board/references/kinds.md` | one short entry per shape of note, read only when a shape is needed |
| `rules/` | each rule an agent keeps to, once: the commands and what the board writes itself (`touching-the-board.md`), how to write and word a note (`writing-notes.md`) |
| `commands/<name>.md` | the slash commands `/board:<name>`: a line that names the skill to follow |
| `harness.go`, `internal/harness` | the same skills carried in the program, for MCP prompts and `stickypane guide` |
| `hooks/` | hook registrations for each agent; both run the scripts below |
| `scripts/` | shell that the hooks and commands run |

The plugin is layered: a command names a skill, a skill points to the rules
and, when needed, to its references, and `using-the-board` is the way in
that names every other skill. A skill opens with what to run, so an agent
can act after reading its first screen; then the rules that are its own.

Each rule is written once, in `rules/`; skills, commands and these docs
point to it. The guide `stickypane init` writes into `AGENTS.md` is
`internal/initcmd/guide.md` followed by the two rule files, so an agent
without the plugin reads the same words; together they stay at most 55
lines. Over MCP, a skill comes with the rules it points to, but not its
references: a shape in full is `stickypane help <shape>` or the guide with
the shape as its topic. `go test ./internal/initcmd/` checks that the skills
stay short, name only files that exist and run only real commands.

The slash commands are `/board:status` (what is on the board, or the notes
that match), `/board:track`, `/board:ask`, `/board:say` and `/board:setup`.
Showing a file is `stickypane show`, and a shape is `stickypane help
<shape>`; neither needs a command of its own.

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
