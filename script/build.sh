#!/usr/bin/env bash
set -euo pipefail
source "$(dirname "${BASH_SOURCE[0]}")/toolchain.sh"
cd "$PROJECT_DIR"
ensure_wails
mkdir -p frontend/dist
if [[ ! -f frontend/dist/index.html ]]; then printf '<!doctype html><title>SkillView</title>' > frontend/dist/index.html; fi
go mod tidy
wails3 generate bindings -ts -i -f '-tags production'
if [[ ! -d frontend/node_modules ]]; then (cd frontend && npm ci); fi
(cd frontend && npm run build)
python3 script/notices.py
APP_BUNDLE="${SKILLVIEW_APP_BUNDLE:-$PROJECT_DIR/bin/SkillView.app}"
mkdir -p "$APP_BUNDLE/Contents/MacOS" "$APP_BUNDLE/Contents/Resources"
go build -tags production -trimpath -buildvcs=false -ldflags='-w -s' -o "$APP_BUNDLE/Contents/MacOS/SkillView" .
cp build/Info.plist "$APP_BUNDLE/Contents/Info.plist"
python3 - "$APP_BUNDLE/Contents/Info.plist" <<'PY'
import pathlib
import plistlib
import sys
version = pathlib.Path('internal/updates/version.txt').read_text().strip()
path = pathlib.Path(sys.argv[1])
info = plistlib.loads(path.read_bytes())
info['CFBundleShortVersionString'] = version.split('-')[0]
info['CFBundleVersion'] = version.split('-')[0]
info['SkillViewReleaseVersion'] = version
path.write_bytes(plistlib.dumps(info))
PY
cp LICENSE "$APP_BUNDLE/Contents/Resources/License.txt"
python3 script/localizations.py "$APP_BUNDLE/Contents/Resources"
for LANGUAGE_FILE in "$APP_BUNDLE"/Contents/Resources/*.lproj/InfoPlist.strings; do
  plutil -lint "$LANGUAGE_FILE"
done
bash script/icon.sh
cp "$SKILLVIEW_TMP_DIR/AppIcon.icns" "$APP_BUNDLE/Contents/Resources/AppIcon.icns"
if [[ -f THIRD_PARTY_NOTICES.md ]]; then cp THIRD_PARTY_NOTICES.md "$APP_BUNDLE/Contents/Resources/ThirdPartyNotices.txt"; fi
if [[ -n "${SIGNING_IDENTITY:-}" ]]; then
  codesign --force --options runtime --timestamp --sign "$SIGNING_IDENTITY" "$APP_BUNDLE"
else
  codesign --force --sign - "$APP_BUNDLE"
fi
codesign --verify --strict --verbose=2 "$APP_BUNDLE"
