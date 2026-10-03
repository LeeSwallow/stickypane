---
description: Show the shapes a note on the stickypane board can take, or one of them in full
allowed-tools: Bash(stickypane:*)
---

Run `stickypane kinds $ARGUMENTS` and show its output to the user as it is,
in a code block. With no argument it lists the shapes; with one (`board`,
`form`, ...) it prints what that shape is for, how it works and an example.
If the user asked to see it on the board, write the example into the
notes folder and run `stickypane show <name>`.
