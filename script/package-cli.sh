#!/usr/bin/env bash
set -euo pipefail
source "$(dirname "${BASH_SOURCE[0]}")/toolchain.sh"
cd "$PROJECT_DIR"
if [[ "${1:-}" != "--skip-build" ]]; then bash script/build-cli.sh; fi
VERSION=$(tr -d '\r\n' < internal/updates/version.txt)
CLI_BINARY="${SKILLVIEW_CLI_BINARY:-$PROJECT_DIR/bin/skillview}"
RELEASE_DIR="${SKILLVIEW_RELEASE_DIR:-$PROJECT_DIR/bin/releases}"
PACKAGE="skillview-cli-$VERSION-macos-arm64.tar.gz"
[[ "$(lipo -archs "$CLI_BINARY")" == arm64 ]]
codesign --verify --strict "$CLI_BINARY"
mkdir -p "$RELEASE_DIR" "$SKILLVIEW_TMP_DIR"
STAGING=$(mktemp -d "$SKILLVIEW_TMP_DIR/package-cli.XXXXXX")
trap 'rm -rf "$STAGING"' EXIT
cp "$CLI_BINARY" "$STAGING/skillview"
cp LICENSE THIRD_PARTY_NOTICES.md "$STAGING/"
printf 'Version: %s\nSource commit: %s\nPlatform: macos-arm64\n' \
  "$VERSION" "$(git rev-parse HEAD)" > "$STAGING/BUILD_INFO.txt"
COPYFILE_DISABLE=1 tar -czf "$RELEASE_DIR/$PACKAGE" -C "$STAGING" \
  skillview LICENSE THIRD_PARTY_NOTICES.md BUILD_INFO.txt
cp script/install-cli.sh "$RELEASE_DIR/install-cli.sh"
(cd "$RELEASE_DIR" && shasum -a 256 "$PACKAGE" > "$PACKAGE.sha256")
echo "Created $RELEASE_DIR/$PACKAGE"
