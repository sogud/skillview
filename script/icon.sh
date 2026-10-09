#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "${BASH_SOURCE[0]}")/.."
mkdir -p .artifacts/AppIcon.iconset
if [[ ! -x .artifacts/icon-renderer || build/icon.swift -nt .artifacts/icon-renderer ]]; then
  xcrun swiftc build/icon.swift -o .artifacts/icon-renderer
fi
.artifacts/icon-renderer .artifacts/AppIcon.iconset
iconutil -c icns .artifacts/AppIcon.iconset -o .artifacts/AppIcon.icns
