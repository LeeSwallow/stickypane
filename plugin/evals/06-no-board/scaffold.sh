#!/usr/bin/env bash
# Runs in the case's empty git repository before the agent starts.
set -eu
git commit -q --allow-empty -m init
# No board: a .sticky/ after the run is a change.
printf "groceries: milk, eggs\n" > notes.txt
