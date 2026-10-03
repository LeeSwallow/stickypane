---
description: Talk with the user in a chat note on the board, and listen for the reply
allowed-tools: Bash(stickypane:*)
---

Follow the `board:talking-in-chat` skill. Say this in the chat note `chat`:
$ARGUMENTS

Run `stickypane say chat "<message>" --as claude`, then wait for a reply
with `stickypane watch --once --note chat --type message.added --timeout 10m --json`
and act on it. Exit code 3: no reply came; say so.
