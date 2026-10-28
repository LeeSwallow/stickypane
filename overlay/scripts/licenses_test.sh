#!/bin/sh
# licenses_test.sh: licenses.sh fails on a package with a license that is not
# allowed and names it, and passes when an allowed alternative exists.
set -u
here="$(cd "$(dirname "$0")" && pwd)"
fail=0

out="$("$here/licenses.sh" --no-go --npm-dir "$here/testdata/gpl" 2>&1)"
code=$?
if [ "$code" -ne 1 ]; then
	echo "FAIL: GPL fixture exited $code, want 1" >&2
	fail=1
fi
case "$out" in
*fake-gpl*GPL-3.0*) ;;
*) echo "FAIL: GPL fixture output does not name fake-gpl: $out" >&2; fail=1 ;;
esac

if ! "$here/licenses.sh" --no-go --npm-dir "$here/testdata/mit" >/dev/null 2>&1; then
	echo "FAIL: '(GPL-3.0 OR MIT)' fixture should pass" >&2
	fail=1
fi

[ "$fail" -eq 0 ] && echo "licenses_test: ok"
exit $fail
