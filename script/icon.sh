#!/usr/bin/env bash
set -euo pipefail
source "$(dirname "${BASH_SOURCE[0]}")/toolchain.sh"
cd "$PROJECT_DIR"
mkdir -p "$SKILLVIEW_TMP_DIR/AppIcon.iconset"
if [[ ! -x "$SKILLVIEW_TMP_DIR/icon-renderer" || build/icon.swift -nt "$SKILLVIEW_TMP_DIR/icon-renderer" ]]; then
  xcrun swiftc build/icon.swift -o "$SKILLVIEW_TMP_DIR/icon-renderer"
fi
"$SKILLVIEW_TMP_DIR/icon-renderer" "$SKILLVIEW_TMP_DIR/AppIcon.iconset"
iconutil -c icns "$SKILLVIEW_TMP_DIR/AppIcon.iconset" -o "$SKILLVIEW_TMP_DIR/AppIcon.icns"
