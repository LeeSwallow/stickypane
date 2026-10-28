#!/bin/sh
# tray.sh rebuilds the tray icons from their sources:
#   tray-template.svg -> tray-template.png (18x18) and tray-template@2x.png
#   (36x36), the macOS template image (black and alpha only);
#   ../appicon.png    -> ../tray.png (44x44), the colour icon for Linux.
# It needs rsvg-convert (librsvg) and sips (macOS).
set -eu
cd "$(dirname "$0")"
rsvg-convert -w 18 -h 18 tray-template.svg -o tray-template.png
rsvg-convert -w 36 -h 36 tray-template.svg -o tray-template@2x.png
sips -z 44 44 ../appicon.png --out ../tray.png >/dev/null
