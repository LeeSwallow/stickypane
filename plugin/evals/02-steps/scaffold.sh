#!/usr/bin/env bash
# Runs in the case's empty git repository before the agent starts.
set -eu
git commit -q --allow-empty -m init
mkdir -p .sticky
for n in 1 2 3; do printf "echo step %s done\n" "$n" > "step$n.sh"; done
