#!/bin/sh
# overlay-linux.sh packs the desktop app for Linux (docs/overlay/09 §9.1):
#
#	sh scripts/overlay-linux.sh <stickypane-app binary> <version> <goarch> <outdir>
#
# writes <outdir>/stickypane-app-<version>-linux-<goarch>.tar.gz holding one
# folder of the same name: stickypane-app (the app, from `wails3 build` on
# Linux), stickypane (the command line program, built here from this
# checkout with the same version, which the app finds next to itself),
# stickypane.desktop, stickypane.png, LICENSE and THIRD_PARTY_NOTICES.md.
# The app needs WebKitGTK 4.1. It leaves no files of its own behind.
set -eu

die() { echo "overlay-linux: $*" >&2; exit 1; }

[ $# -eq 4 ] || die "usage: overlay-linux.sh <stickypane-app binary> <version> <goarch> <outdir>"
bin=$1 version=$2 goarch=$3 outdir=$4
[ -f "$bin" ] && [ -x "$bin" ] || die "$bin: not an executable"
case $goarch in amd64 | arm64) ;; *) die "$goarch: want amd64 or arm64" ;; esac

root=$(cd "$(dirname "$0")/.." && pwd)
pkg="stickypane-app-$version-linux-$goarch"
work=$(mktemp -d "${TMPDIR:-/tmp}/overlay-linux.XXXXXX")
trap 'rm -rf "$work"' EXIT

dir="$work/$pkg"
mkdir -p "$dir"
cp "$bin" "$dir/stickypane-app"
(cd "$root" && CGO_ENABLED=0 GOOS=linux GOARCH=$goarch go build -trimpath \
	-ldflags "-s -w -X main.version=$version" -o "$dir/stickypane" ./cmd/stickypane)
cp "$root/overlay/build/linux/stickypane.desktop" "$dir/"
cp "$root/overlay/build/appicon.png" "$dir/stickypane.png"
cp "$root/LICENSE" "$root/THIRD_PARTY_NOTICES.md" "$dir/"

mkdir -p "$outdir"
out="$(cd "$outdir" && pwd)/$pkg.tar.gz"
tar -C "$work" -czf "$out" "$pkg"
echo "overlay-linux: wrote $out"
