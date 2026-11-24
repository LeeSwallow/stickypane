#!/usr/bin/env sh
# At the start of a session, tells the agent that the project has a board
# and what is on it. Says nothing when there is no board or no stickypane,
# so a project without one costs no context.
set -u
here="$(cd "$(dirname "$0")" && pwd)"
# The session's folder is the cwd of the hook's input, not
# CLAUDE_PROJECT_DIR, which stays at the project's root in a worktree.
# Read with sed, as jq may be missing: a path with a quote or a backslash
# in it falls back to $PWD.
[ -t 0 ] || cwd="$(sed -n 's/.*"cwd"[[:space:]]*:[[:space:]]*"\([^"\\]*\)".*/\1/p' | head -1)"
[ -n "${cwd:-}" ] && cd "$cwd" 2>/dev/null
context="$("$here/board-context.sh" 2>/dev/null)" || exit 0
dir="$(printf '%s\n' "$context" | sed -n 's/^BOARD_DIR=//p')"
notes="$(printf '%s\n' "$context" | sed -n '/^NOTES<<END$/,/^END$/p' | sed '1d;$d' | head -20)"
printf 'The user keeps a stickypane board open next to you (%s). On it now:\n%s\n' "$dir" "${notes:-(nothing yet)}"
printf 'Put things there with the board:using-the-board skill: `stickypane show <file>`, `stickypane todo|card|chart|log`, or a Markdown file in that folder.\n'
# In a linked worktree, its folder on the board, as the board names it.
if wt="$(stickypane worktree show 2>/dev/null)"; then
  printf '%s\n' "$wt" | sed -n 's/^name: //p;s/^folder: //p' | {
    read -r name
    read -r folder
    printf 'worktree: %s → %s\n' "$name" "$folder"
  }
fi
