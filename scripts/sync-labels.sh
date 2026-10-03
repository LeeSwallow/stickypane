#!/bin/sh
# sync-labels.sh [owner/repo]: create or update the labels in
# .github/labels.txt on GitHub, with the gh CLI. Labels that are not in the
# file are left alone.
set -eu
repo="${1:-LeeSwallow/stickypane}"
file="$(dirname "$0")/../.github/labels.txt"
grep -v '^#' "$file" | grep -v '^$' | while IFS='|' read -r name color desc; do
	gh label create "$name" --repo "$repo" --color "$color" --description "$desc" --force >/dev/null
	echo "label: $name"
done
