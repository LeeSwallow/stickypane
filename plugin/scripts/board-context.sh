#!/usr/bin/env sh
# Prints what an agent needs to know about the board of the project in the
# current directory, as KEY=VALUE lines, then the list of notes.
#
# Exit codes: 0 a board was found; 3 no board at or above the current
# directory, nor in the main worktree of its repository; 4 stickypane is
# not installed.
set -u

if ! command -v stickypane >/dev/null 2>&1; then
  printf 'STICKYPANE_INSTALLED=no\n'
  exit 4
fi
printf 'STICKYPANE_INSTALLED=yes\nSTICKYPANE_VERSION=%s\n' "$(stickypane version 2>/dev/null | awk '{print $2}')"

found() {
  printf 'BOARD_DIR=%s\n' "$1"
  printf 'NOTES<<END\n%s\nEND\n' "$(stickypane note list 2>/dev/null)"
  exit 0
}

dir="${PWD:-.}"
while [ "$dir" != "/" ]; do
  for name in .sticky; do
    [ -d "$dir/$name" ] && found "$dir/$name"
  done
  parent="$(dirname "$dir")"
  # A path dirname cannot shorten (".", or a Windows path) ends the walk.
  [ "$parent" = "$dir" ] && break
  dir="$parent"
done
# .sticky/ is gitignored, so a linked worktree has none of its own: its
# board is the main worktree's, the first one git lists.
main="$(git -C "${PWD:-.}" worktree list --porcelain 2>/dev/null | sed -n '1s/^worktree //p')"
[ -n "$main" ] && [ -d "$main/.sticky" ] && found "$main/.sticky"
printf 'BOARD_DIR=\n'
exit 3
