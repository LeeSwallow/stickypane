#!/usr/bin/env sh
# Tells the agent, at the start of a session, whether the project has a
# stickypane board and what is on it. Prints nothing when there is none, so
# a project without a board costs no context.
set -u

if ! command -v stickypane >/dev/null 2>&1; then
  exit 0
fi

dir="${PWD:-.}"
while [ "$dir" != "/" ]; do
  if [ -d "$dir/.sticky" ] || [ -d "$dir/.stickypane" ]; then
    notes="$(stickypane list 2>/dev/null | head -20)"
    printf 'The user keeps a stickypane board open next to you (folder: %s). Notes on it now:\n%s\n' \
      "$([ -d "$dir/.sticky" ] && echo "$dir/.sticky" || echo "$dir/.stickypane")" "${notes:-(none yet)}"
    printf 'Use the board:using-the-board skill to put things there: `stickypane show <file>`, `stickypane todo/card/chart/log`, or write a Markdown file into the folder.\n'
    exit 0
  fi
  dir="$(dirname "$dir")"
done
exit 0
