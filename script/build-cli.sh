#!/usr/bin/env bash
set -euo pipefail
source "$(dirname "${BASH_SOURCE[0]}")/toolchain.sh"
cd "$PROJECT_DIR"
if [[ "$(uname -s)-$(uname -m)" != "Darwin-arm64" ]]; then
  echo "The standalone CLI currently supports macOS Apple Silicon only." >&2
  exit 1
fi
ensure_go
VERSION=$(tr -d '\r\n' < internal/updates/version.txt)
CLI_BINARY="${SKILLVIEW_CLI_BINARY:-$PROJECT_DIR/bin/skillview}"
mkdir -p "$(dirname "$CLI_BINARY")"
CGO_ENABLED=1 GOOS=darwin GOARCH=arm64 go build -trimpath -buildvcs=false \
  -ldflags="-w -s -X main.version=$VERSION" -o "$CLI_BINARY" ./cmd/skillview
if [[ -n "${SIGNING_IDENTITY:-}" ]]; then
  codesign --force --options runtime --timestamp --sign "$SIGNING_IDENTITY" "$CLI_BINARY"
else
  codesign --force --sign - "$CLI_BINARY"
fi
codesign --verify --strict "$CLI_BINARY"
echo "Built $CLI_BINARY"
