#!/usr/bin/env sh
# Prints what an agent needs to know about the board of the project in the
# current directory, as KEY=VALUE lines, then the list of notes.
#
# Exit codes: 0 a board was found; 3 no board at or above the current
# directory; 4 stickypane is not installed.
set -u

if ! command -v stickypane >/dev/null 2>&1; then
  printf 'STICKYPANE_INSTALLED=no\n'
  exit 4
fi
printf 'STICKYPANE_INSTALLED=yes\nSTICKYPANE_VERSION=%s\n' "$(stickypane version 2>/dev/null | awk '{print $2}')"

dir="${PWD:-.}"
while [ "$dir" != "/" ]; do
  for name in .sticky .stickypane; do
    if [ -d "$dir/$name" ]; then
      printf 'BOARD_DIR=%s\n' "$dir/$name"
      printf 'NOTES<<END\n%s\nEND\n' "$(stickypane list 2>/dev/null)"
      exit 0
    fi
  done
  dir="$(dirname "$dir")"
done
printf 'BOARD_DIR=\n'
exit 3
