---
description: Ask the user a question on the board, as a form with buttons, and wait for the answer
allowed-tools: Bash(stickypane:*), Write
---

Ask this on the board, following the `board:asking-the-user` skill:
$ARGUMENTS

Write the form into `.sticky/`, run `stickypane show <name>`, then
`stickypane wait <name> --timeout 10m`. Report the answer, or that the time
ran out.
