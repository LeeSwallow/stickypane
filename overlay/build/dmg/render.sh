#!/bin/sh
# render.sh draws the DMG window's background from background.html with
# headless Chrome: background.png (660x420), background@2x.png (1320x840),
# and background.tiff, the two in one file for Finder (docs/overlay/09 §9.2).
# It needs Google Chrome, the frontend's node_modules (npm ci in
# overlay/frontend) for the fonts and rough.js, and tiffutil (macOS).
#
#	sh overlay/build/dmg/render.sh
set -eu
here=$(cd "$(dirname "$0")" && pwd)
chrome=${CHROME:-"/Applications/Google Chrome.app/Contents/MacOS/Google Chrome"}
[ -x "$chrome" ] || { echo "render.sh: no Chrome at $chrome (set CHROME)" >&2; exit 1; }
[ -f "$here/../../frontend/node_modules/roughjs/bundled/rough.js" ] ||
	{ echo "render.sh: run npm ci in overlay/frontend first" >&2; exit 1; }

profile=$(mktemp -d "${TMPDIR:-/tmp}/stickypane-dmg-chrome.XXXXXX")
trap 'rm -rf "$profile"' EXIT

shot() { # shot <scale> <out.png>
	"$chrome" --headless=new --disable-gpu --hide-scrollbars \
		--user-data-dir="$profile" --no-first-run --no-default-browser-check \
		--allow-file-access-from-files --virtual-time-budget=4000 \
		--window-size=660,420 --force-device-scale-factor="$1" \
		--screenshot="$2" "file://$here/background.html" >/dev/null 2>&1
	[ -s "$2" ] || { echo "render.sh: Chrome made no $2" >&2; exit 1; }
}
shot 1 "$here/background.png"
shot 2 "$here/background@2x.png"
tiffutil -cathidpicheck "$here/background.png" "$here/background@2x.png" \
	-out "$here/background.tiff" >/dev/null 2>&1
echo "render.sh: wrote background.png, background@2x.png, background.tiff"
