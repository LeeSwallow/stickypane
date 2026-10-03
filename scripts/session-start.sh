#!/usr/bin/env sh
# At the start of a session, tells the agent that the project has a board
# and what is on it. Says nothing when there is no board or no stickypane,
# so a project without one costs no context.
set -u
here="$(cd "$(dirname "$0")" && pwd)"
context="$("$here/board-context.sh" 2>/dev/null)" || exit 0
dir="$(printf '%s\n' "$context" | sed -n 's/^BOARD_DIR=//p')"
notes="$(printf '%s\n' "$context" | sed -n '/^NOTES<<END$/,/^END$/p' | sed '1d;$d' | head -20)"
printf 'The user keeps a stickypane board open next to you (folder: %s). Notes on it now:\n%s\n' "$dir" "${notes:-(none yet)}"
printf 'To put something there use the board:using-the-board skill: `stickypane show <file>`, `stickypane todo|card|chart|log`, or write a Markdown file into the folder.\n'
