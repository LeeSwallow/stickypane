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
| `mods/claude/` | `board-mod`, a second plugin in the same marketplace: a Claude Code mod (TypeScript hooks) that shows the board in a pane and the user's changes as toasts |

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

## The mod

`mods/claude/` is a Claude Code mod, installed on its own
(`/plugin install board-mod --marketplace LeeSwallow/stickypane`). Its
hooks module, `hooks/register.tsx`, reads the board only through the
command line: `stickypane index --json` for the pane, `stickypane cat` for
a note, and `stickypane watch --json` for the session's life. An event
that arrives while one of the agent's tool calls runs, or within 1.5 s
after one ends, is taken as the agent's and not announced. What it makes
of those lines is in `hooks/board.ts`, which touches no engine API.

```sh
claude plugin validate mods/claude    # what the engine would refuse
claude plugin test mods/claude        # hooks/*.test.ts(x)
claude --plugin-dir mods/claude       # a session with the mod loaded
```

The types it is checked against are written by Claude Code into
`mods/claude/.claude-plugin/types/` when it loads the mod; that folder is
not committed.

