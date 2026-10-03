---
description: Set this project up for the stickypane board
allowed-tools: Bash(stickypane:*), Bash(command -v stickypane), Bash(go install:*), Bash(brew install:*)
---

1. If `stickypane` is not installed, say how: `brew install --cask
   LeeSwallow/tap/stickypane` or `go install
   github.com/LeeSwallow/stickypane/cmd/stickypane@latest`, and stop.
2. Run `stickypane init --no-agent-docs`. This plugin already teaches the
   board, so the guide is not added to AGENTS.md.
3. Tell the user to open the board next to you: `tmux split-window -h
   stickypane` (or `stickypane` in any other pane), and that `stickypane
   show README.md` puts a first thing on it.
