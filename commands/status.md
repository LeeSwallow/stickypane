---
description: What is on the stickypane board right now
allowed-tools: Bash(${CLAUDE_PLUGIN_ROOT}/scripts/board-context.sh), Bash(stickypane:*)
---

## The board

!`${CLAUDE_PLUGIN_ROOT}/scripts/board-context.sh`

## To do

Say in a few lines what the user has open and what the agent-facing notes
(checklists, boards, forms) stand at. Exit code 3 means there is no board:
offer `/board:setup`. Exit code 4 means stickypane is not installed: say
how to install it. Do not read every file; use the list, and
`stickypane cat <name>` only for a note the user asks about.
