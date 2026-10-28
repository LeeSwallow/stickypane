#!/bin/sh
# dmg_test.sh: scripts/overlay-dmg.sh fails cleanly on a bad .app: it exits
# non-zero, names the problem, writes no image and mounts nothing.
set -u
here="$(cd "$(dirname "$0")" && pwd)"
dmg="$here/../../scripts/overlay-dmg.sh"
tmp="$(mktemp -d "${TMPDIR:-/tmp}/dmg_test.XXXXXX")"
trap 'rm -rf "$tmp"' EXIT
fail=0

mounted() { command -v hdiutil >/dev/null 2>&1 && hdiutil info | grep image-path | sort; }

# expect_fail <label> <want in output> <args...>
expect_fail() {
	label=$1 want=$2
	shift 2
	before="$(mounted)"
	out="$(sh "$dmg" "$@" 2>&1)"
	code=$?
	if [ "$code" -eq 0 ]; then
		echo "FAIL: $label exited 0" >&2
		fail=1
	fi
	case "$out" in
	*"$want"*) ;;
	*) echo "FAIL: $label: output does not say \"$want\": $out" >&2; fail=1 ;;
	esac
	if [ -e "$tmp/out.dmg" ]; then
		echo "FAIL: $label wrote an image" >&2
		fail=1
	fi
	if [ "$(mounted)" != "$before" ]; then
		echo "FAIL: $label left an image mounted" >&2
		fail=1
	fi
}

expect_fail "no arguments" "usage"
expect_fail "missing .app" "not an .app bundle" "$tmp/Missing.app" "$tmp/out.dmg"
mkdir -p "$tmp/Empty.app/Contents"
expect_fail "empty .app" "not an .app bundle" "$tmp/Empty.app" "$tmp/out.dmg"
expect_fail "not a .dmg" "must end in .dmg" "$tmp/Missing.app" "$tmp/out.img"

[ "$fail" -eq 0 ] && echo "dmg_test: ok"
exit $fail
