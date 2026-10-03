#!/usr/bin/env sh
# Repository settings that cannot live in files: run once after the first
# push, by the maintainer, with the GitHub CLI logged in. Usage:
#   scripts/repo-setup.sh OWNER/REPO
set -eu
repo="${1:?usage: repo-setup.sh OWNER/REPO}"

gh repo edit "$repo" \
  --description "A board in your terminal that your coding agent writes to and you read" \
  --homepage "https://github.com/$repo" \
  --enable-discussions \
  --enable-wiki=false \
  --delete-branch-on-merge \
  --add-topic tui --add-topic terminal --add-topic go --add-topic golang --add-topic bubbletea \
  --add-topic cli --add-topic developer-tools --add-topic coding-agents --add-topic ai-agents \
  --add-topic claude-code --add-topic codex --add-topic markdown --add-topic kanban --add-topic dashboard

# The labels: one list, in .github/labels.txt.
sh "$(dirname "$0")/sync-labels.sh" "$repo"
# Security reports reach the maintainer privately, from the Security tab.
gh api -X PUT "repos/$repo/private-vulnerability-reporting" >/dev/null && echo "private vulnerability reporting: on"

printf '\nDone. Still by hand: pin an Ideas post in Discussions titled "What should the board show you?", and open two or three "feedback wanted" issues from ROADMAP.md.\n'
