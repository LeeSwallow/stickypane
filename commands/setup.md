---
description: Set this project up for the stickypane board
allowed-tools: Bash(stickypane:*)
---

Run `stickypane init` in the project's top folder and report its output.
It is safe to run again, and it leaves the instruction files alone when
this plugin is installed.

If the command is not found, stickypane is not installed: give the user
`brew install --cask LeeSwallow/tap/stickypane` (or `go install
github.com/LeeSwallow/stickypane/cmd/stickypane@latest`) and stop.
