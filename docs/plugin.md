# The plugin

This repository is also a Claude Code and Codex marketplace, `stickypane`,
with one plugin, `board`, in the folder `plugin/`. For how to use the
board, see the [README](../README.md); this page is for changing the plugin.

## Layout

The two marketplace files stay at the repository's root, where
`plugin marketplace add LeeSwallow/stickypane` looks for them, and point
at `./plugin`. Everything else the plugin is made of is in `plugin/`, its
root once installed (`${CLAUDE_PLUGIN_ROOT}`); paths below without a
folder are in it.

| Path | Is |
| --- | --- |
| `.claude-plugin/marketplace.json` (root) | the Claude Code marketplace `stickypane`: the plugin `board` at `./plugin` |
| `.agents/plugins/marketplace.json` (root) | the same for Codex |
| `.claude-plugin/plugin.json` | the Claude Code plugin `board` |
| `.codex-plugin/plugin.json` | the same for Codex |
| `skills/<name>/SKILL.md` | a skill: its name, when to use it (the description), then the commands and the rules |
| `skills/using-the-board/references/kinds.md` | one short entry per shape of note, read only when a shape is needed |
| `rules/` | each rule an agent keeps to, once: the commands and what the board writes itself (`touching-the-board.md`), how to write and word a note (`writing-notes.md`) |
| `guide.md` | the head of the agent guide `stickypane init` writes |
| `commands/<name>.md` | the slash commands `/board:<name>`: a line that names the skill to follow |
| `hooks/` | hook registrations for each agent; both run the scripts below |
| `scripts/` | shell that the hooks and commands run |
| `harness.go` (root), `internal/usecase/harness` | the same skills, rules, commands and guide carried in the program, for MCP prompts, `stickypane guide` and `stickypane init` |

`scripts/` at the root is for developing stickypane, not part of the plugin.

The plugin is layered: a command names a skill, a skill points to the rules
and, when needed, to its references, and `using-the-board` is the way in
that names every other skill. A skill opens with what to run, so an agent
can act after reading its first screen; then the rules that are its own.

Each rule is written once, in `rules/`; skills, commands and these docs
point to it. The guide `stickypane init` writes into `AGENTS.md` is
`plugin/guide.md` followed by the two rule files, so an agent
without the plugin reads the same words; together they stay at most 55
lines. Over MCP, a skill comes with the rules it points to, but not its
references: a shape in full is `stickypane help <shape>` or the guide with
the shape as its topic. `go test ./internal/usecase/initcmd/` checks that the skills
stay short, name only files that exist and run only real commands.

The slash commands are `/board:status` (what is on the board, or the notes
that match), `/board:track`, `/board:ask`, `/board:say` and `/board:setup`.
Showing a file is `stickypane show`, and a shape is `stickypane help
<shape>`; neither needs a command of its own.

## Scripts

| Script | Usage | Does |
| --- | --- | --- |
| `plugin/scripts/board-context.sh` | no arguments, in a project | prints `STICKYPANE_INSTALLED`, `STICKYPANE_VERSION`, `BOARD_DIR` and the note list |
| `plugin/scripts/session-start.sh` | run by the SessionStart hook | tells the agent what is on the board; silent without one |
| `scripts/selfcheck.sh` | no arguments | checks manifests, scripts, skills and commands |

Exit codes: `0` fine; `1` selfcheck found a problem (listed with ❌); `3`
no board at or above the current directory; `4` stickypane is not
installed.

## Adding a skill

1. Make `plugin/skills/<name>/SKILL.md` with `name` and a `description` that
   starts with "Use when". Put the commands first and the rules after, in
   under 80 lines.
2. Run `sh scripts/selfcheck.sh` and `go test ./internal/usecase/initcmd/`.
