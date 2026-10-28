#!/bin/sh
# desktop-dmg.sh makes the desktop app's drag-to-install disk image
# (docs/desktop/09 §9.2): a Finder window of 660x420 points on the
# hand-drawn background, "Stickypane.app" at (170,210), an Applications
# alias at (490,210), icons of 112 points, and the app icon on the volume.
#
#	sh scripts/desktop-dmg.sh desktop/bin/stickypane-app.app stickypane-0.1.0-arm64.dmg
#
# The .app comes from `wails3 package` in desktop/; the background from
# desktop/build/dmg/render.sh. The image's copy of the app carries the
# command line program at Contents/Helpers/stickypane, built here
# from this checkout for the app's architectures with the app's version,
# so installing the app is enough (09 §9.1), and Contents/Resources gets
# LICENSE, THIRD_PARTY_NOTICES.md and licenses/ with every bundled package's
# license text. Both are signed ad hoc (codesign -s -), the program first.
# Finder arranges the window through AppleScript, so it runs on a Mac with
# a logged-in user. It overwrites out.dmg, and whatever happens it leaves no
# volume of its own mounted and no files of its own behind.
set -eu

vol="stickypane"
name="Stickypane.app"

die() { echo "desktop-dmg: $*" >&2; exit 1; }

[ $# -eq 2 ] || die "usage: desktop-dmg.sh <app> <out.dmg>"
app=$1
out=$2
case $out in *.dmg) ;; *) die "$out: the output must end in .dmg" ;; esac
[ -d "$app" ] && [ -f "$app/Contents/Info.plist" ] || die "$app: not an .app bundle"
[ "$(uname -s)" = Darwin ] || die "needs macOS (hdiutil, Finder)"
exe=$(/usr/libexec/PlistBuddy -c 'Print :CFBundleExecutable' "$app/Contents/Info.plist" 2>/dev/null) ||
	die "$app: Info.plist has no CFBundleExecutable"
[ -x "$app/Contents/MacOS/$exe" ] || die "$app: Info.plist names $exe, which is not in Contents/MacOS"

root=$(cd "$(dirname "$0")/.." && pwd)
# One version: desktop/build/config.yml's info.version, which the bundle's
# Info.plist repeats (TestVersionsAgree checks the plists in the tree).
want=$(sed -n 's/^[[:space:]][[:space:]]*version:[[:space:]]*"\([^"]*\)".*/\1/p' "$root/desktop/build/config.yml" | head -n 1)
[ -n "$want" ] || die "desktop/build/config.yml has no info.version"
version=$(/usr/libexec/PlistBuddy -c 'Print :CFBundleShortVersionString' "$app/Contents/Info.plist" 2>/dev/null) ||
	die "$app: Info.plist has no CFBundleShortVersionString"
[ "$version" = "$want" ] ||
	die "$app: version $version, but desktop/build/config.yml says $want: rebuild the app or fix the versions"
bg="$root/desktop/build/dmg/background.tiff"
[ -f "$bg" ] || die "$bg missing: run desktop/build/dmg/render.sh"
icns="$app/Contents/Resources/icons.icns"
[ -f "$icns" ] || icns="$root/desktop/build/darwin/icons.icns"
[ -d "/Volumes/$vol" ] && die "a volume named \"$vol\" is mounted: eject it first"

work=$(mktemp -d "${TMPDIR:-/tmp}/desktop-dmg.XXXXXX")
dev=""
cleanup() {
	if [ -n "$dev" ]; then
		hdiutil detach "$dev" -quiet 2>/dev/null ||
			hdiutil detach "$dev" -force -quiet 2>/dev/null || true
	fi
	rm -rf "$work"
}
trap cleanup EXIT
trap 'exit 130' INT TERM HUP

# The folder the image is made from: the app with its own copy of the
# command line program, signed inside out, and the alias.
stage="$work/stage"
mkdir -p "$stage/.background"
ditto "$app" "$stage/$name"
archs=$(lipo -archs "$app/Contents/MacOS/$exe")
cli="$stage/$name/Contents/Helpers/stickypane"
mkdir -p "$(dirname "$cli")"
set --
for a in $archs; do
	case $a in x86_64) goarch=amd64 ;; arm64) goarch=arm64 ;; *) die "$app: unknown architecture $a" ;; esac
	(cd "$root" && CGO_ENABLED=0 GOOS=darwin GOARCH=$goarch go build -trimpath -buildvcs=false \
		-ldflags "-s -w -X main.version=$version" -o "$work/stickypane-$a" ./cmd/stickypane)
	set -- "$@" "$work/stickypane-$a"
done
lipo -create "$@" -output "$cli"
# The license texts, before signing: the seal covers Resources.
res="$stage/$name/Contents/Resources"
cp "$root/LICENSE" "$root/THIRD_PARTY_NOTICES.md" "$res/"
sh "$root/desktop/scripts/licenses.sh" --write "$res/licenses/stickypane-app.txt" >/dev/null ||
	die "desktop/scripts/licenses.sh failed: run it to see why"
sh "$root/scripts/licenses.sh" "$res/licenses/stickypane.txt" >/dev/null ||
	die "scripts/licenses.sh failed: run it to see why"
xattr -cr "$stage/$name"
codesign --force --sign - "$cli"
codesign --force --sign - "$stage/$name"
codesign --verify --deep --strict "$stage/$name"
ln -s /Applications "$stage/Applications"
cp "$bg" "$stage/.background/background.tiff"

size=$(du -sm "$stage" | cut -f1)
hdiutil create -quiet -volname "$vol" -srcfolder "$stage" -fs HFS+ \
	-format UDRW -size "$((size + 20))m" -ov "$work/rw.dmg"
# The mount point is the one hdiutil reports (tab-separated: device, type,
# mount point); Finder addresses the disk by name, so it must be ours.
line=$(hdiutil attach -readwrite -noverify -noautoopen "$work/rw.dmg" | grep 'Apple_HFS' | head -n 1) ||
	die "could not attach the image"
dev=$(printf '%s\n' "$line" | awk '{ print $1 }')
mnt=$(printf '%s\n' "$line" | awk -F '\t' '{ print $NF }' | sed 's/[[:space:]]*$//')
[ -n "$dev" ] || die "could not attach the image"
[ "$mnt" = "/Volumes/$vol" ] || die "the image mounted at \"$mnt\", not /Volumes/$vol"

# Finder lays the window out and writes it to the volume's .DS_Store.
# The bounds are the window's frame: 660x420 of content under a title bar.
if ! osascript - "$vol" "$name" <<'EOF'
on run argv
	set vol to item 1 of argv
	set appName to item 2 of argv
	tell application "Finder"
		tell disk vol
			open
			set current view of container window to icon view
			set toolbar visible of container window to false
			set statusbar visible of container window to false
			set the bounds of container window to {200, 120, 860, 568}
			set opts to the icon view options of container window
			set arrangement of opts to not arranged
			set icon size of opts to 112
			set text size of opts to 13
			set label position of opts to bottom
			set shows item info of opts to false
			set shows icon preview of opts to true
			set background picture of opts to file ".background:background.tiff"
			set position of item appName of container window to {170, 210}
			set position of item "Applications" of container window to {490, 210}
			update without registering applications
			delay 1
			close
		end tell
	end tell
end run
EOF
then
	die "osascript could not arrange the Finder window. If macOS denied it, allow your terminal to control Finder in System Settings > Privacy & Security > Automation, then run this again"
fi

# Wait for Finder to write .DS_Store, then mark the volume icon and ask
# Finder to open the window when the image is mounted. bless --openfolder
# is not supported on Apple Silicon, so a failure there is only a warning.
i=0
while [ ! -f "$mnt/.DS_Store" ] && [ $i -lt 20 ]; do sleep 0.5; i=$((i + 1)); done
[ -f "$mnt/.DS_Store" ] || die "Finder wrote no .DS_Store"
# hdiutil -srcfolder leaves .VolumeIcon.icns out, so it goes in here.
cp "$icns" "$mnt/.VolumeIcon.icns"
SetFile -a C "$mnt" 2>/dev/null || echo "desktop-dmg: SetFile missing; no volume icon" >&2
bless --folder "$mnt" --openfolder "$mnt" ||
	echo "desktop-dmg: warning: bless --openfolder failed; the window will not open on mount" >&2
chmod -Rf go-w "$mnt" 2>/dev/null || true
rm -rf "$mnt/.fseventsd" 2>/dev/null || true
sync

hdiutil detach "$dev" -quiet || hdiutil detach "$dev" -force -quiet
dev=""
mkdir -p "$(dirname "$out")"
rm -f "$out"
hdiutil convert -quiet "$work/rw.dmg" -format UDZO -imagekey zlib-level=9 -o "$out"
echo "desktop-dmg: wrote $out ($(du -h "$out" | cut -f1))"
