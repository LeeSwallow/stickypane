#!/bin/sh
# issue.sh <number>: start work on a GitHub issue. It makes a branch from
# the latest main named after the issue, such as fix/12-show-readme, so the
# commit-msg hook can add "Refs: #12" to every commit on it and pr.sh can
# open the pull request that closes it.
set -eu
n="${1:?usage: scripts/issue.sh <issue number>}"
repo="${REPO:-LeeSwallow/stickypane}"
info="$(gh issue view "$n" --repo "$repo" --json title,labels,state --jq '[.state, .title, ([.labels[].name] | join(","))] | join("\t")')"
state="$(printf '%s' "$info" | cut -f1)"
title="$(printf '%s' "$info" | cut -f2)"
labels="$(printf '%s' "$info" | cut -f3)"
if [ "$state" != "OPEN" ]; then
	echo "issue #$n is $state" >&2
	exit 1
fi
# The kind of branch follows the issue's label.
case ",$labels," in
*,bug,*) kind=fix ;;
*,docs,*) kind=docs ;;
*,chore,*) kind=chore ;;
*) kind=feat ;;
esac
slug="$(printf '%s' "$title" | sed -E 's/^\[[a-z]+\] *//' | tr '[:upper:]' '[:lower:]' | sed -E 's/[^a-z0-9]+/-/g; s/^-+//; s/-+$//' | cut -c1-40 | sed -E 's/-+$//')"
branch="$kind/$n-${slug:-work}"
git fetch --quiet origin main
git switch -c "$branch" origin/main
echo "On $branch for #$n: $title"
echo "Commit as usual; open the pull request with scripts/pr.sh."
