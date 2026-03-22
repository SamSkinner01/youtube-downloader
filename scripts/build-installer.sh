#!/usr/bin/env bash
set -e

APP_NAME="YouTube Downloader"
BUNDLE_ID="com.youtube-downloader.app"
ICON_SRC="../assets/ytd.jpg"
APP_BUNDLE="${APP_NAME}.app"

echo "==> Building binary..."
VERSION=$(git describe --tags --exact-match 2>/dev/null | sed 's/^v//' || echo "${VERSION:-dev}")
DMG_NAME="YouTube-Downloader-${VERSION}.dmg"
go build \
  -ldflags "-X youtube-downloader/internal/version.Current=${VERSION}" \
  -o youtube-downloader-gui \
  ../cmd/gui/

echo "==> Creating icon..."
ICONSET="AppIcon.iconset"
rm -rf "$ICONSET"
mkdir "$ICONSET"

for size in 16 32 64 128 256 512; do
  sips -z $size $size "$ICON_SRC" --setProperty format png --out "${ICONSET}/icon_${size}x${size}.png"     >/dev/null
  double=$((size * 2))
  sips -z $double $double "$ICON_SRC" --setProperty format png --out "${ICONSET}/icon_${size}x${size}@2x.png" >/dev/null
done

iconutil -c icns "$ICONSET" -o AppIcon.icns
rm -rf "$ICONSET"

echo "==> Assembling .app bundle..."
rm -rf "$APP_BUNDLE"
mkdir -p "${APP_BUNDLE}/Contents/MacOS"
mkdir -p "${APP_BUNDLE}/Contents/Resources"

cp youtube-downloader-gui "${APP_BUNDLE}/Contents/MacOS/${APP_NAME}"
cp AppIcon.icns "${APP_BUNDLE}/Contents/Resources/AppIcon.icns"

cat > "${APP_BUNDLE}/Contents/Info.plist" <<PLIST
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN"
  "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
  <key>CFBundleName</key>
  <string>${APP_NAME}</string>
  <key>CFBundleDisplayName</key>
  <string>${APP_NAME}</string>
  <key>CFBundleIdentifier</key>
  <string>${BUNDLE_ID}</string>
  <key>CFBundleVersion</key>
  <string>${VERSION}</string>
  <key>CFBundleShortVersionString</key>
  <string>${VERSION}</string>
  <key>CFBundleExecutable</key>
  <string>${APP_NAME}</string>
  <key>CFBundleIconFile</key>
  <string>AppIcon</string>
  <key>CFBundlePackageType</key>
  <string>APPL</string>
  <key>NSHighResolutionCapable</key>
  <true/>
  <key>LSMinimumSystemVersion</key>
  <string>11.0</string>
</dict>
</plist>
PLIST

echo "==> Signing .app bundle..."
if [ -n "$APPLE_SIGN_IDENTITY" ]; then
  # Full Developer ID signing (requires Apple Developer account)
  codesign --sign "$APPLE_SIGN_IDENTITY" \
    --deep --force --options runtime \
    --entitlements ../scripts/entitlements.plist \
    "$APP_BUNDLE"
  echo "    Signed with: $APPLE_SIGN_IDENTITY"
else
  # Ad-hoc signing — lets the app run on the build machine without Gatekeeper prompts.
  # Distributed builds will still show a Gatekeeper warning unless notarized.
  codesign --sign - --deep --force "$APP_BUNDLE"
  echo "    Ad-hoc signed (no Developer ID found)"
fi

echo "==> Creating DMG..."
STAGING="dmg-staging"
rm -rf "$STAGING"
mkdir "$STAGING"
cp -r "$APP_BUNDLE" "$STAGING/"
ln -s /Applications "$STAGING/Applications"

rm -f "$DMG_NAME"
hdiutil create \
  -volname "${APP_NAME}" \
  -srcfolder "$STAGING" \
  -ov \
  -format UDZO \
  "$DMG_NAME"

rm -rf "$STAGING" AppIcon.icns

echo ""
echo "Done: ${DMG_NAME}"
