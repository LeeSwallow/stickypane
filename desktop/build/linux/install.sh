#!/bin/sh
# install.sh installs stickypane for this user, under ~/.local, from the
# folder it is in (the unpacked stickypane-app-<version>-linux-<arch>):
#
#	sh install.sh             install or update
#	sh install.sh --uninstall remove what it installed
#
# ~/.local/lib/stickypane-app/     stickypane-app and the stickypane it runs
# ~/.local/bin/stickypane-app      a link to the app
# ~/.local/share/applications/stickypane.desktop
# ~/.local/share/icons/hicolor/512x512/apps/stickypane.png
#
# The app needs GTK 4 and WebKitGTK 6.0 (libgtk-4-1 and libwebkitgtk-6.0-4 on Debian and Ubuntu).
set -eu

here=$(cd "$(dirname "$0")" && pwd)
data=${XDG_DATA_HOME:-$HOME/.local/share}
lib="$HOME/.local/lib/stickypane-app"
bin="$HOME/.local/bin"
desktop="$data/applications/stickypane.desktop"
icon="$data/icons/hicolor/512x512/apps/stickypane.png"

if [ "${1:-}" = --uninstall ]; then
	rm -rf "$lib"
	rm -f "$bin/stickypane-app" "$desktop" "$icon"
	echo "stickypane: removed from ~/.local"
	exit 0
fi
[ $# -eq 0 ] || { echo "usage: sh install.sh [--uninstall]" >&2; exit 2; }

mkdir -p "$lib" "$bin" "$(dirname "$desktop")" "$(dirname "$icon")"
cp "$here/stickypane-app" "$here/stickypane" "$lib/"
cp -R "$here/LICENSE" "$here/THIRD_PARTY_NOTICES.md" "$here/licenses" "$lib/"
ln -sf "$lib/stickypane-app" "$bin/stickypane-app"
# A desktop session may not have ~/.local/bin on its PATH: name the app by path.
sed "s|^Exec=stickypane-app|Exec=$lib/stickypane-app|" "$here/stickypane.desktop" >"$desktop"
cp "$here/stickypane.png" "$icon"
command -v update-desktop-database >/dev/null 2>&1 && update-desktop-database -q "$data/applications" || true
echo "stickypane: installed in $lib; run stickypane-app or open it from the applications menu"
