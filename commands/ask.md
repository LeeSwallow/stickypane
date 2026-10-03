---
description: Ask the user a question on the board, as a form with buttons, and wait for the answer
allowed-tools: Bash(stickypane:*), Write
---

Turn this into a form on the board and wait for the answer, following the
`board:asking-the-user` skill: $ARGUMENTS

Write the form to the notes folder, `stickypane show` it, then
`stickypane wait <name> --timeout 10m`. Report the answer, or that the
time ran out.
