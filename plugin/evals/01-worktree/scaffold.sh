#!/usr/bin/env bash
# Runs in the case's empty git repository before the agent starts.
set -eu
git commit -q --allow-empty -m init
mkdir -p .sticky
git worktree add -q wt/feat-chart -b feat-chart
echo wt >> .git/info/exclude
# The runner starts the session in the worktree named here.
echo wt/feat-chart > .eval-cwd
