#!/bin/sh
# desktop-linux.sh packs the desktop app for Linux (docs/desktop/09 §9.1):
#
#	sh scripts/desktop-linux.sh <stickypane-app binary> <version> <goarch> <outdir>
#
# writes <outdir>/stickypane-app-<version>-linux-<goarch>.tar.gz holding one
# folder of the same name: stickypane-app (the app, from `wails3 build` on
# Linux), stickypane (the command line program, built here from this
# checkout with the same version, which the app finds next to itself),
# stickypane.desktop, stickypane.png, install.sh (which installs it all under
# ~/.local), LICENSE, THIRD_PARTY_NOTICES.md and licenses/ (every bundled
# package's license text).
# The app needs WebKitGTK 4.1. It leaves no files of its own behind.
set -eu

die() { echo "desktop-linux: $*" >&2; exit 1; }

[ $# -eq 4 ] || die "usage: desktop-linux.sh <stickypane-app binary> <version> <goarch> <outdir>"
bin=$1 version=$2 goarch=$3 outdir=$4
[ -f "$bin" ] && [ -x "$bin" ] || die "$bin: not an executable"
case $goarch in amd64 | arm64) ;; *) die "$goarch: want amd64 or arm64" ;; esac

root=$(cd "$(dirname "$0")/.." && pwd)
pkg="stickypane-app-$version-linux-$goarch"
work=$(mktemp -d "${TMPDIR:-/tmp}/desktop-linux.XXXXXX")
trap 'rm -rf "$work"' EXIT

dir="$work/$pkg"
mkdir -p "$dir"
cp "$bin" "$dir/stickypane-app"
(cd "$root" && CGO_ENABLED=0 GOOS=linux GOARCH=$goarch go build -trimpath -buildvcs=false \
	-ldflags "-s -w -X main.version=$version" -o "$dir/stickypane" ./cmd/stickypane)
cp "$root/desktop/build/linux/stickypane.desktop" "$root/desktop/build/linux/install.sh" "$dir/"
cp "$root/desktop/build/appicon.png" "$dir/stickypane.png"
cp "$root/LICENSE" "$root/THIRD_PARTY_NOTICES.md" "$dir/"
sh "$root/desktop/scripts/licenses.sh" --write "$dir/licenses/stickypane-app.txt" >/dev/null ||
	die "desktop/scripts/licenses.sh failed: run it to see why"
sh "$root/scripts/licenses.sh" "$dir/licenses/stickypane.txt" >/dev/null ||
	die "scripts/licenses.sh failed: run it to see why"

mkdir -p "$outdir"
out="$(cd "$outdir" && pwd)/$pkg.tar.gz"
# GNU tar makes the archive reproducible: sorted names, owner and group 0,
# and every time SOURCE_DATE_EPOCH (else the last commit's). Another tar
# makes an ordinary one.
if tar --version 2>/dev/null | grep -q 'GNU tar'; then
	epoch=${SOURCE_DATE_EPOCH:-$(git -C "$root" log -1 --format=%ct 2>/dev/null || echo 0)}
	tar -C "$work" --sort=name --owner=0 --group=0 --numeric-owner \
		--mtime="@$epoch" -cf - "$pkg" | gzip -n >"$out"
else
	tar -C "$work" -czf "$out" "$pkg"
fi
echo "desktop-linux: wrote $out"
