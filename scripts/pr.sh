#!/bin/sh
# pr.sh: push the branch made by issue.sh and open its pull request. The
# body closes the issue, the kind label comes from the issue, and the title
# is the first commit's subject unless you give one:
#
#   scripts/pr.sh ["feat: what changes"]
set -eu
repo="${REPO:-LeeSwallow/stickypane}"
branch="$(git rev-parse --abbrev-ref HEAD)"
n="$(printf '%s' "$branch" | sed -nE 's|^[a-z]+/([0-9]+)-.*|\1|p')"
if [ -z "$n" ]; then
	echo "$branch is not an issue branch; start one with scripts/issue.sh <number>" >&2
	exit 1
fi
title="${1:-$(git log --reverse --format=%s origin/main..HEAD | head -1)}"
labels="$(gh issue view "$n" --repo "$repo" --json labels --jq '[.labels[].name | select(. == "feature" or . == "enhancement" or . == "bug" or . == "breaking change" or . == "docs" or . == "plugin" or . == "chore")] | join(",")')"
case "$branch" in
fix/*) default=bug ;;
docs/*) default=docs ;;
chore/*) default=chore ;;
*) default=enhancement ;;
esac
sh "$(dirname "$0")/check-commit-trailers.sh" "origin/main..HEAD"
git push -u origin "$branch"
body="Closes #$n

$(git log --reverse --format='- %s' origin/main..HEAD)"
gh pr create --repo "$repo" --base main --head "$branch" --title "$title" --body "$body" --label "${labels:-$default}"
