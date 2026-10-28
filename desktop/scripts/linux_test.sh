#!/bin/sh
# linux_test.sh: scripts/desktop-linux.sh packs the Linux archive of
# docs/desktop/09 §9.1 — one folder holding the app, the command line
# program, the .desktop entry, the icon, install.sh and the licenses — and
# that folder's install.sh installs it under ~/.local and removes it again.
# It runs on any Unix with Go: the app is a stand-in script, HOME a
# temporary folder, and it refuses bad arguments without writing anything.
set -u
here="$(cd "$(dirname "$0")" && pwd)"
linux="$here/../../scripts/desktop-linux.sh"
tmp="$(mktemp -d "${TMPDIR:-/tmp}/linux_test.XXXXXX")"
trap 'rm -rf "$tmp"' EXIT
fail=0
no() { echo "FAIL: $*" >&2; fail=1; }

# expect_fail <label> <want in output> <args...>
expect_fail() {
	label=$1 want=$2
	shift 2
	out="$(sh "$linux" "$@" 2>&1)" && no "$label exited 0"
	case "$out" in *"$want"*) ;; *) no "$label: output does not say \"$want\": $out" ;; esac
	[ -z "$(ls -A "$tmp/bad" 2>/dev/null)" ] || no "$label wrote into the output folder"
}

printf '#!/bin/sh\necho stand-in\n' >"$tmp/stickypane-app"
chmod +x "$tmp/stickypane-app"
printf 'not a program\n' >"$tmp/plain"
expect_fail "no arguments" "usage"
expect_fail "missing app" "not an executable" "$tmp/missing" 1.2.3 amd64 "$tmp/bad"
expect_fail "app not executable" "not an executable" "$tmp/plain" 1.2.3 amd64 "$tmp/bad"
expect_fail "unknown arch" "want amd64 or arm64" "$tmp/stickypane-app" 1.2.3 386 "$tmp/bad"

pkg=stickypane-app-1.2.3-linux-amd64
sh "$linux" "$tmp/stickypane-app" 1.2.3 amd64 "$tmp/out" >/dev/null || no "desktop-linux.sh failed"
tgz="$tmp/out/$pkg.tar.gz"
[ -f "$tgz" ] || { no "no $pkg.tar.gz"; exit 1; }

# The layout, file by file: one folder, nothing else.
got="$(tar -tzf "$tgz" | sed 's|/$||' | grep -v "^$pkg/licenses/." | sort)"
want="$(printf '%s\n' "$pkg" "$pkg/LICENSE" "$pkg/THIRD_PARTY_NOTICES.md" "$pkg/install.sh" \
	"$pkg/licenses" "$pkg/stickypane" "$pkg/stickypane-app" "$pkg/stickypane.desktop" \
	"$pkg/stickypane.png" | sort)"
[ "$got" = "$want" ] || no "archive layout:
got:
$got
want:
$want"
lic="$(tar -tzf "$tgz" | grep "^$pkg/licenses/." | sort | tr '\n' ' ')"
[ "$lic" = "$pkg/licenses/stickypane-app.txt $pkg/licenses/stickypane.txt " ] ||
	no "licenses/ holds: $lic"

mkdir "$tmp/x"
tar -xzf "$tgz" -C "$tmp/x"
d="$tmp/x/$pkg"
for f in stickypane-app stickypane; do
	[ -x "$d/$f" ] || no "$f is not executable"
done
[ "$(head -c 4 "$d/stickypane" | od -An -c | tr -d ' ')" = '177ELF' ] || no "stickypane is not a Linux program"
cmp -s "$d/stickypane-app" "$tmp/stickypane-app" || no "stickypane-app is not the app given"
cmp -s "$d/stickypane.png" "$here/../build/appicon.png" || no "stickypane.png is not the app icon"
grep -qx 'Exec=stickypane-app %u' "$d/stickypane.desktop" || no ".desktop does not run stickypane-app"
grep -qx 'Icon=stickypane' "$d/stickypane.desktop" || no ".desktop does not name the icon stickypane"
grep -qx 'Name=stickypane' "$d/stickypane.desktop" || no ".desktop is not named stickypane"

# install.sh, into a HOME of its own.
h="$tmp/home"
mkdir "$h"
HOME="$h" XDG_DATA_HOME= sh "$d/install.sh" >/dev/null || no "install.sh failed"
lib="$h/.local/lib/stickypane-app"
for f in "$lib/stickypane-app" "$lib/stickypane" "$lib/LICENSE" "$lib/THIRD_PARTY_NOTICES.md" \
	"$lib/licenses/stickypane-app.txt" "$h/.local/share/icons/hicolor/512x512/apps/stickypane.png"; do
	[ -f "$f" ] || no "install.sh did not install ${f#"$h"/}"
done
[ "$(readlink "$h/.local/bin/stickypane-app")" = "$lib/stickypane-app" ] || no "no ~/.local/bin/stickypane-app link"
grep -qx "Exec=$lib/stickypane-app %u" "$h/.local/share/applications/stickypane.desktop" ||
	no "the installed .desktop does not name the app by path"
HOME="$h" XDG_DATA_HOME= sh "$d/install.sh" --uninstall >/dev/null || no "install.sh --uninstall failed"
left="$(find "$h" -type f -o -type l)"
[ -z "$left" ] || no "install.sh --uninstall left: $left"

[ "$fail" -eq 0 ] && echo "linux_test: ok"
exit $fail
