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

label() { gh label create "$1" --repo "$repo" --color "$2" --description "$3" --force >/dev/null && printf 'label %s\n' "$1"; }
label "design"          "7057ff" "How the board should behave; wanted at this stage"
label "feedback wanted" "0e8a16" "The maintainer wants to hear from users before deciding"
label "needs decision"  "fbca04" "Blocked on a design decision"
label "good first issue" "7057ff" "Small, well described, a good way in"
label "help wanted"     "008672" "The maintainer would welcome a hand"
label "needs info"      "d876e3" "Waiting for details from the reporter"
label "waiting on op"   "d876e3" "Waiting for the person who opened it"
label "planned"         "1d76db" "On the roadmap"
label "breaking change" "b60205" "Changes files, names or keys users rely on"
label "feature"         "a2eeef" "Something new (release notes: New)"
label "enhancement"     "a2eeef" "Something better (release notes: Changed)"
label "bug"             "d73a4a" "Something broken (release notes: Fixed)"
label "docs"            "0075ca" "README, guide, skills (release notes: Docs and plugin)"
label "plugin"          "0075ca" "The Claude Code / Codex plugin"
label "skip-changelog"  "ededed" "Left out of the release notes"

printf '\nDone. Still by hand: pin an Ideas post in Discussions titled "What should the board show you?", and open two or three "feedback wanted" issues from ROADMAP.md.\n'
