#!/bin/sh
# live-setup.sh dir: a small board for demo/live.tape, in a new project at
# dir, opened in tmux beside a shell.
set -eu
mkdir -p "$1" && cd "$1"
git init -q
export STICKYPANE_USER=sam
sp() { stickypane "$@" >/dev/null; }
stickypane init --no-agent-docs >/dev/null
rm -f .sticky/welcome.md
sp theme catppuccin-mocha
sp language en
printf -- '---\ntype: board\n---\n## To do\n\n## Doing\n\n## Done\n' | stickypane write work --open >/dev/null
for c in "login API @claude" "rate limits" "password reset"; do sp card work add "$c" --to "To do"; done
sp card work move "login" --to Doing
sp set work title="Auth sprint" size=half
for i in "read the issue" "add the endpoint" "write the tests" "open the PR"; do sp todo plan add "$i"; done
sp todo plan check "read the issue"
sp set plan title="Plan" size=half
# The agent's shell on the left, the board on the right.
tmux -L live kill-server 2>/dev/null || true
exec tmux -L live -f /dev/null new -s live "env PS1='agent \$ ' bash --norc" \; \
	set status off \; split-window -h -l 60% stickypane \; select-pane -L
