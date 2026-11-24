#!/usr/bin/env bash
# Runs in the case's empty git repository before the agent starts.
set -eu
git commit -q --allow-empty -m init
mkdir -p .sticky
mkdir -p .cache/build && touch .cache/build/a.o
