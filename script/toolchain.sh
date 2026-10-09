#!/usr/bin/env bash
set -euo pipefail

PROJECT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
TOOLS_DIR="$PROJECT_DIR/.artifacts/toolchain"
export GOPATH="$PROJECT_DIR/.artifacts/go"
export GOCACHE="$PROJECT_DIR/.artifacts/go-cache"
export GOBIN="$TOOLS_DIR/bin"
export PATH="$TOOLS_DIR/go/bin:$GOBIN:$PATH"
export MACOSX_DEPLOYMENT_TARGET=13.0
export CGO_CFLAGS="-mmacosx-version-min=13.0 ${CGO_CFLAGS:-}"
export CGO_LDFLAGS="-mmacosx-version-min=13.0 ${CGO_LDFLAGS:-}"

ensure_go() {
  if command -v go >/dev/null 2>&1; then return; fi
  if [[ "$(uname -s)-$(uname -m)" != "Darwin-arm64" ]]; then
    echo "Install Go 1.25 or newer for this platform before building SkillView." >&2
    return 1
  fi
  mkdir -p "$TOOLS_DIR"
  local archive="$TOOLS_DIR/go1.27.1.darwin-arm64.tar.gz"
  curl --fail --location --retry 2 https://go.dev/dl/go1.27.1.darwin-arm64.tar.gz -o "$archive"
  echo "ee215d57e0ec269c60cc9ceca68e6bda321ba9ee5afe24f4b0988703c2d87d12  $archive" | shasum -a 256 -c -
  tar -xzf "$archive" -C "$TOOLS_DIR"
}

ensure_wails() {
  ensure_go
  if [[ ! -x "$GOBIN/wails3" ]]; then
    go install github.com/wailsapp/wails/v3/cmd/wails3@v3.0.0-beta.27
  fi
}

if [[ "${BASH_SOURCE[0]}" == "$0" ]]; then
  ensure_wails
  go version
  wails3 version
fi
