#!/bin/sh
# check-pr.sh: check that a pull request follows the flow in CONTRIBUTING.md.
# It reads PR_TITLE, PR_BODY, PR_LABELS (one per line) and PR_AUTHOR from the
# environment, so it runs the same in CI and by hand.
#
#   1. The title is a Conventional Commit, since it becomes the commit on
#      main when the pull request is squashed: "fix: show README.md".
#   2. The body closes the issue it comes from ("Closes #12"), unless the
#      change is small enough for the no-issue label. Bots are exempt.
#   3. It has one kind label, which puts it in the right part of the release
#      notes (.github/release.yml).
set -u
fail=0
say() { echo "✗ $*"; fail=1; }

title="${PR_TITLE:-}"
body="${PR_BODY:-}"
labels="${PR_LABELS:-}"
author="${PR_AUTHOR:-}"
has_label() { printf '%s\n' "$labels" | grep -qxF "$1"; }

if ! printf '%s' "$title" | grep -qE '^(feat|fix|perf|refactor|docs|test|build|ci|chore|revert)(\([a-z0-9/._-]+\))?!?: .+'; then
	say "the title \"$title\" is not a Conventional Commit: type(scope): what changes, as in \"fix: show README.md links a root file\""
fi

case "$author" in
*\[bot\] | dependabot*) ;;
*)
	if ! has_label no-issue && ! printf '%s' "$body" | grep -qiE '(close[sd]?|fix(e[sd])?|resolve[sd]?) +(([a-z0-9_.-]+/[a-z0-9_.-]+)?#[0-9]+)'; then
		say "the body names no issue it closes: add \"Closes #<issue>\", or the no-issue label for a small fix"
	fi
	;;
esac

kinds=0
for k in feature enhancement bug "breaking change" docs plugin chore dependencies; do
	if has_label "$k"; then kinds=$((kinds + 1)); fi
done
if [ "$kinds" -eq 0 ]; then
	say "no kind label: add one of feature, enhancement, bug, breaking change, docs, plugin, chore"
fi

if [ "$fail" -eq 0 ]; then
	echo "✓ the pull request follows the flow"
fi
exit "$fail"
