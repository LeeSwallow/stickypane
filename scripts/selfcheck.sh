#!/usr/bin/env sh
# Checks the plugin's own files: manifests parse, scripts are valid shell,
# every skill has a name and a description and every command a description.
# The Go tests check the rest (go test ./internal/initcmd/). Exit code 1 when
# any check fails.
set -u
here="$(cd "$(dirname "$0")/.." && pwd)"
fail=0
ok() { printf '✅ %s\n' "$1"; }
bad() { printf '❌ %s\n' "$1"; fail=1; }

for f in .claude-plugin/plugin.json .claude-plugin/marketplace.json .codex-plugin/plugin.json .agents/plugins/marketplace.json hooks/claude-hooks.json hooks/codex-hooks.json; do
  if python3 -c "import json,sys; json.load(open(sys.argv[1]))" "$here/$f" 2>/dev/null; then ok "$f parses"; else bad "$f is not valid JSON"; fi
done
for f in scripts/*.sh; do
  if sh -n "$here/$f" 2>/dev/null; then ok "$f is valid shell"; else bad "$f has a syntax error"; fi
done
for d in "$here"/skills/*/; do
  s="$d/SKILL.md"
  name="$(basename "$d")"
  if grep -q "^name: $name$" "$s" && grep -q '^description: ' "$s"; then ok "skill $name has its name and description"; else bad "skill $name: SKILL.md needs 'name: $name' and a description"; fi
done
for f in "$here"/commands/*.md; do
  head -1 "$f" | grep -q '^---$' && sed -n 2,4p "$f" | grep -q '^description: ' && ok "command $(basename "$f") has a description" || bad "command $(basename "$f") needs front matter with a description"
done
exit $fail
