---
description: What is on the stickypane board right now
allowed-tools: Bash(${CLAUDE_PLUGIN_ROOT}/scripts/board-context.sh), Bash(stickypane:*)
---

!`${CLAUDE_PLUGIN_ROOT}/scripts/board-context.sh`

Say in a few lines what is open on the board and where the checklists,
boards and forms stand, from the list above. Read a note with
`stickypane cat <name>` only when the user asks about it.
`STICKYPANE_INSTALLED=no`: say how to install it (`brew install --cask
LeeSwallow/tap/stickypane`). An empty `BOARD_DIR`: there is no board yet;
the first `stickypane show` or `todo` makes one.
