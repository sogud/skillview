#!/usr/bin/env bash
set -euo pipefail
source "$(dirname "${BASH_SOURCE[0]}")/toolchain.sh"
cd "$PROJECT_DIR"
if [[ $# != 2 ]]; then
  echo 'Usage: script/sign-update.sh <reviewed SkillView ZIP> <release notes file>' >&2
  exit 1
fi
ZIP="$1"
NOTES="$2"
PRIVATE_KEY="${SKILLVIEW_UPDATE_PRIVATE_KEY:-}"
if [[ ! -f "$ZIP" || ! -f "$NOTES" || ! -f "$PRIVATE_KEY" ]]; then
  echo 'Provide the reviewed ZIP, release notes, and SKILLVIEW_UPDATE_PRIVATE_KEY path.' >&2
  exit 1
fi
ensure_wails
VERSION=$(tr -d '\r\n' < internal/updates/version.txt)
if [[ "$(basename "$ZIP")" != "SkillView-$VERSION-macos-arm64.zip" ]]; then
  echo 'ZIP filename must match the release version and supported architecture.' >&2
  exit 1
fi
OUTPUT="$(dirname "$ZIP")/manifest.json"
# Sign only the reviewed ZIP. The private key is never copied into the app,
# repository, manifest, or build artifacts.
wails3 updater manifest -version "$VERSION" -key "$PRIVATE_KEY" \
  -name "SkillView $VERSION" -notes-file "$NOTES" \
  -url-prefix "https://github.com/sogud/skillview/releases/download/v$VERSION" \
  -output "$OUTPUT" "$ZIP"
wails3 updater verify -manifest "$OUTPUT" -publickey internal/updates/updater.key.pub
