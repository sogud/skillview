#!/usr/bin/env bash
set -euo pipefail

PROJECT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
TOOLS_DIR="$PROJECT_DIR/runtime/toolchain"
export SKILLVIEW_TMP_DIR="${SKILLVIEW_TMP_DIR:-$PROJECT_DIR/tmp}"
export GOPATH="$PROJECT_DIR/runtime/go"
export GOCACHE="$SKILLVIEW_TMP_DIR/go-cache"
export GOBIN="$TOOLS_DIR/bin"
export PATH="$TOOLS_DIR/go/bin:$GOBIN:$PATH"
# Keep downloaded Go sources outside the application module during recursive checks.
mkdir -p "$PROJECT_DIR/runtime"
if [[ ! -f "$PROJECT_DIR/runtime/go.mod" ]]; then
  printf "module skillview-runtime\n" > "$PROJECT_DIR/runtime/go.mod"
fi
export MACOSX_DEPLOYMENT_TARGET=13.0
export CGO_CFLAGS="-mmacosx-version-min=13.0 ${CGO_CFLAGS:-}"
export CGO_LDFLAGS="-mmacosx-version-min=13.0 ${CGO_LDFLAGS:-}"

ensure_go() {
  if command -v go >/dev/null 2>&1 && [[ "$(go env GOVERSION)" == "go1.27.3" ]]; then return; fi
  if [[ "$(uname -s)-$(uname -m)" != "Darwin-arm64" ]]; then
    echo "Install Go 1.27.3 for this platform before building SkillView." >&2
    return 1
  fi
  mkdir -p "$TOOLS_DIR"
  local archive="$TOOLS_DIR/go1.27.3.darwin-arm64.tar.gz"
  curl --fail --location --retry 2 https://go.dev/dl/go1.27.3.darwin-arm64.tar.gz -o "$archive"
  echo "dd3908cf20959d798c1e06ebe5b72213f0505ce01373b40ce3110975a6cd99c2  $archive" | shasum -a 256 -c -
  local staging
  staging=$(mktemp -d "$TOOLS_DIR/go-install.XXXXXX")
  tar -xzf "$archive" -C "$staging"
  # Extract into a fresh directory; mixing Go releases can break the toolchain.
  if [[ -d "$TOOLS_DIR/go" ]]; then mv "$TOOLS_DIR/go" "$staging/previous-go"; fi
  mv "$staging/go" "$TOOLS_DIR/go"
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
