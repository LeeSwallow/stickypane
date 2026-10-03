---
description: Set this project up for the stickypane board
allowed-tools: Bash(${CLAUDE_PLUGIN_ROOT}/scripts/board-context.sh), Bash(stickypane:*)
---

## Where things stand

!`${CLAUDE_PLUGIN_ROOT}/scripts/board-context.sh`

## To do

1. `STICKYPANE_INSTALLED=no`: say how to install it — `brew install --cask
   LeeSwallow/tap/stickypane` or `go install
   github.com/LeeSwallow/stickypane/cmd/stickypane@latest` — and stop.
2. `BOARD_DIR` is empty: run `stickypane init --no-agent-docs`. This plugin
   teaches the board, so the guide is not added to AGENTS.md.
3. Tell the user to open the board next to you (`tmux split-window -h
   stickypane`, or `stickypane` in any pane) and that
   `stickypane show README.md` puts a first thing on it.
