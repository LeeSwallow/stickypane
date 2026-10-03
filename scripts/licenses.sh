#!/bin/sh
# licenses.sh: collect the license of every module linked into the program,
# for the release archives. It writes them into one file, by default
# build/THIRD_PARTY_LICENSES.txt, and fails when a module has no license file
# or one that is not MIT, BSD or Apache-2.0, so a dependency with other terms
# cannot slip into a release unnoticed.
#
#   scripts/licenses.sh [out-file]
set -eu
here="$(cd "$(dirname "$0")/.." && pwd)"
out="${1:-$here/build/THIRD_PARTY_LICENSES.txt}"
mkdir -p "$(dirname "$out")"
cd "$here"
fail=0
{
	echo "stickypane is MIT licensed (LICENSE). It links the modules below; each"
	echo "license is reproduced as its module ships it."
	go list -deps -f '{{with .Module}}{{.Path}} {{.Version}} {{.Dir}}{{end}}' ./cmd/stickypane | sort -u |
		while read -r path version dir; do
			[ "$path" = github.com/LeeSwallow/stickypane ] && continue
			file=""
			for f in "$dir"/LICENSE "$dir"/LICENSE.txt "$dir"/LICENSE.md "$dir"/LICENCE "$dir"/COPYING; do
				if [ -f "$f" ]; then file="$f"; break; fi
			done
			if [ -z "$file" ]; then
				echo "no license file in $path" >&2
				exit 1
			fi
			if ! grep -qE 'Permission is hereby granted|Redistribution and use|Apache License' "$file"; then
				echo "$path is not MIT, BSD or Apache-2.0: read $file before releasing" >&2
				exit 1
			fi
			printf '\n\n================================================================\n%s %s\n================================================================\n\n' "$path" "$version"
			cat "$file"
		done
} >"$out" || fail=1
[ "$fail" -eq 0 ] && echo "wrote $out"
exit $fail
