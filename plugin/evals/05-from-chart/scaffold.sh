#!/usr/bin/env bash
# Runs in the case's empty git repository before the agent starts.
set -eu
git commit -q --allow-empty -m init
stickypane todo add api "list the routes" >/dev/null
stickypane todo add api "write the handlers" >/dev/null
stickypane todo add api "test the handlers" >/dev/null
stickypane todo check api routes >/dev/null
stickypane todo check api "write the" >/dev/null
stickypane todo add ui "draw the page" >/dev/null
stickypane todo add ui "wire the form" >/dev/null
stickypane todo add ui "style it" >/dev/null
stickypane todo add ui "test it" >/dev/null
stickypane todo check ui draw >/dev/null
