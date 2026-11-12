#!/bin/sh
# desktop-linux-packages.sh makes the app's rpm and deb (docs/desktop/09
# §9.1) from the Linux archive desktop-linux.sh wrote:
#
#	sh scripts/desktop-linux-packages.sh <stickypane-app-<version>-linux-<goarch>.tar.gz> <outdir>
#
# writes <outdir>/stickypane-app-<version>-linux-<goarch>.rpm and .deb. They
# hold what the archive holds, laid out by desktop/build/linux/nfpm/nfpm.yaml.
# It needs wails3, whose `tool package` is nfpm.
set -eu

die() { echo "desktop-linux-packages: $*" >&2; exit 1; }

[ $# -eq 2 ] || die "usage: desktop-linux-packages.sh <archive.tar.gz> <outdir>"
tgz=$1 outdir=$2
[ -f "$tgz" ] || die "$tgz: no such archive"
pkg=$(basename "$tgz" .tar.gz)
rest=${pkg#stickypane-app-}
goarch=${rest##*-linux-}
version=${rest%-linux-*}
case $goarch in amd64 | arm64) ;; *) die "$tgz: not named stickypane-app-<version>-linux-<amd64|arm64>.tar.gz" ;; esac
[ -n "$version" ] && [ "$version" != "$rest" ] || die "$tgz: not named stickypane-app-<version>-linux-<amd64|arm64>.tar.gz"
command -v wails3 >/dev/null 2>&1 || die "wails3 is not on the PATH"

root=$(cd "$(dirname "$0")/.." && pwd)
mkdir -p "$outdir"
outdir=$(cd "$outdir" && pwd)
tgz="$(cd "$(dirname "$tgz")" && pwd)/$(basename "$tgz")"
work=$(mktemp -d "${TMPDIR:-/tmp}/desktop-linux-packages.XXXXXX")
stage="$root/desktop/bin/pkg"
trap 'rm -rf "$work" "$stage"' EXIT

# nfpm.yaml names its files from desktop/, so the archive is unpacked there.
rm -rf "$stage"
mkdir -p "$stage"
tar -xzf "$tgz" -C "$stage" --strip-components=1
cd "$root/desktop"
for format in rpm deb; do
	mkdir "$work/$format"
	VERSION=$version GOARCH=$goarch wails3 tool package -name stickypane-app -format "$format" \
		-config ./build/linux/nfpm/nfpm.yaml -out "$work/$format" >/dev/null ||
		die "wails3 tool package -format $format failed"
	made=$(find "$work/$format" -type f -name "*.$format")
	[ -n "$made" ] && [ "$(printf '%s\n' "$made" | wc -l)" -eq 1 ] || die "expected one .$format, got: $made"
	mv "$made" "$outdir/$pkg.$format"
	echo "desktop-linux-packages: wrote $outdir/$pkg.$format"
done
