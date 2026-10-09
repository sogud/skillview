#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "${BASH_SOURCE[0]}")/.."
if [[ "${1:-}" != "--skip-build" ]]; then bash script/build.sh; fi
APP_BUNDLE="${SKILLVIEW_APP_BUNDLE:-$PWD/bin/SkillView.app}"
RELEASE_DIR="${SKILLVIEW_RELEASE_DIR:-$PWD/bin/releases}"
VERSION=$(/usr/libexec/PlistBuddy -c 'Print CFBundleShortVersionString' "$APP_BUNDLE/Contents/Info.plist")
ARCH=$(lipo -archs "$APP_BUNDLE/Contents/MacOS/SkillView" | tr ' ' '-')
PACKAGE="SkillView-$VERSION-macos-$ARCH"
mkdir -p "$RELEASE_DIR"
plutil -lint "$APP_BUNDLE/Contents/Info.plist"
codesign --verify --strict --verbose=2 "$APP_BUNDLE"
STAGING=$(mktemp -d "$PWD/.artifacts/package.XXXXXX")
trap 'rm -rf "$STAGING"' EXIT
ditto "$APP_BUNDLE" "$STAGING/SkillView.app"
ditto -c -k --sequesterRsrc --keepParent "$STAGING/SkillView.app" "$RELEASE_DIR/$PACKAGE.zip"
ln -s /Applications "$STAGING/Applications"
hdiutil create -volname SkillView -srcfolder "$STAGING" -format UDZO -ov "$RELEASE_DIR/$PACKAGE.dmg"
(cd "$RELEASE_DIR" && shasum -a 256 "$PACKAGE.zip" "$PACKAGE.dmg" > "$PACKAGE.sha256")
echo "Created packages in $RELEASE_DIR. Ad-hoc signed builds are experimental previews; production distribution requires Developer ID signing and Apple notarization."
