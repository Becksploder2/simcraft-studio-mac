#!/bin/bash
set -euo pipefail
cd "$(dirname "$0")"
if ! command -v xcrun >/dev/null 2>&1; then
  echo "This build script must run on a Mac with Xcode Command Line Tools installed."
  exit 1
fi
APP="Simscraft Studio.app"
mkdir -p "$APP/Contents/MacOS" "$APP/Contents/Resources"
cp -R Resources/. "$APP/Contents/Resources/"
cp Info.plist "$APP/Contents/Info.plist"
iconutil -c icns AppIcon.iconset -o "$APP/Contents/Resources/AppIcon.icns"
xcrun swiftc -parse-as-library -framework SwiftUI -framework AppKit -O CASCraftMac.swift -o "$APP/Contents/MacOS/SimscraftStudio"
chmod +x "$APP/Contents/MacOS/SimscraftStudio"
echo "Built $APP. Open it to test before sharing."
