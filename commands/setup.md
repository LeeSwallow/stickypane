---
description: Install the stickypane board plugin for this machine's agents and set this project up
allowed-tools: Bash(stickypane:*)
---

Run `stickypane setup --dry-run` and show the user the plan it prints. Ask
which scope they want (user: every project; project: this repository,
shared; local: this repository, just them) and whether to register the MCP
server too, then run `stickypane setup --yes` with `--scope <scope>` and,
if they want it, `--mcp`. Report what it printed.

If the command is not found, stickypane is not installed: give the user
`brew install --cask LeeSwallow/tap/stickypane` (or `go install
github.com/LeeSwallow/stickypane/cmd/stickypane@latest`) and stop.
