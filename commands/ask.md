---
description: Ask the user a question on the board, as a form with buttons, and wait for the answer
allowed-tools: Bash(stickypane:*), Write
---

Follow the `board:asking-the-user` skill to ask this on the board:
$ARGUMENTS

Write the form into `.sticky/`, run `stickypane show <name>`, then
`stickypane wait <name> --timeout 10m`. Report the answer, or that the time
ran out.
