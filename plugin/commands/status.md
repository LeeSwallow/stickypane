---
description: What is on the stickypane board right now, or the notes that match
allowed-tools: Bash(${CLAUDE_PLUGIN_ROOT}/scripts/board-context.sh), Bash(stickypane:*)
---

!`${CLAUDE_PLUGIN_ROOT}/scripts/board-context.sh`

`STICKYPANE_INSTALLED=no`: say how to install it (`brew install --cask
LeeSwallow/tap/stickypane`) and stop. An empty `BOARD_DIR`: there is no
board yet; the first `stickypane show` or `todo` makes one.

Otherwise run `stickypane index` and say in a few lines what is on the
board, by tab, and where the checklists, boards and forms stand. With an
argument, show only the lines that match `$ARGUMENTS`. Read a note with
`stickypane cat <name>` only when the user asks about it.
