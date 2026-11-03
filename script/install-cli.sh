#!/usr/bin/env bash
set -euo pipefail

usage() {
  echo "Usage: install-cli.sh [--version <tag>] [--bin-dir <path>] [--force] [--dry-run]"
}

version="latest"
bin_dir="${HOME:?HOME must be set}/.local/bin"
force=false
dry_run=false
while [[ $# -gt 0 ]]; do
  case "$1" in
    --version|--bin-dir)
      [[ $# -ge 2 && -n "$2" && "$2" != --* ]] || { usage >&2; exit 2; }
      if [[ "$1" == --version ]]; then version="$2"; else bin_dir="$2"; fi
      shift 2 ;;
    --force) force=true; shift ;;
    --dry-run) dry_run=true; shift ;;
    --help|-h) usage; exit 0 ;;
    *) usage >&2; exit 2 ;;
  esac
done

if [[ "$(uname -s)-$(uname -m)" != "Darwin-arm64" ]]; then
  echo "The standalone CLI currently supports macOS Apple Silicon only." >&2
  exit 1
fi
if [[ "$version" != latest && ! "$version" =~ ^v?[0-9]+\.[0-9]+\.[0-9]+(-[A-Za-z0-9.-]+)?$ ]]; then
  echo "Use a release version such as v0.1.4." >&2
  exit 2
fi
target="$bin_dir/skillview"
if [[ -e "$target" || -L "$target" ]]; then
  if [[ "$force" != true || -d "$target" ]]; then
    echo "Command already exists: $target. Inspect it before using --force." >&2
    exit 1
  fi
fi
if [[ "$dry_run" == true ]]; then
  echo "Would install SkillView CLI ($version, macos-arm64) to $target; no files changed."
  exit 0
fi

curl_args=(--proto '=https' --proto-redir '=https' --tlsv1.2 --fail --location --silent --show-error --retry 2)
if [[ "$version" == latest ]]; then
  release_url=$(curl "${curl_args[@]}" --head --output /dev/null --write-out '%{url_effective}' \
    https://github.com/sogud/skillview/releases/latest)
  version="${release_url##*/}"
fi
if [[ ! "$version" =~ ^v?[0-9]+\.[0-9]+\.[0-9]+(-[A-Za-z0-9.-]+)?$ ]]; then
  echo "Could not resolve a valid release version." >&2
  exit 1
fi
version="v${version#v}"
asset="skillview-cli-${version#v}-macos-arm64.tar.gz"
url="https://github.com/sogud/skillview/releases/download/$version/$asset"
staging=$(mktemp -d)
pending=""
trap 'rm -rf "$staging"; if [[ -n "$pending" ]]; then rm -f "$pending"; fi' EXIT
curl "${curl_args[@]}" "$url" --output "$staging/$asset"
curl "${curl_args[@]}" "$url.sha256" --output "$staging/$asset.sha256"
if ! read -r expected filename < "$staging/$asset.sha256" || [[ ! "$expected" =~ ^[a-fA-F0-9]{64}$ || "$filename" != "$asset" ]]; then
  echo "Invalid release checksum file; nothing installed." >&2
  exit 1
fi
actual=$(shasum -a 256 "$staging/$asset")
actual="${actual%% *}"
if [[ "$actual" != "$expected" ]]; then
  echo "Release checksum mismatch; nothing installed." >&2
  exit 1
fi
# Extract only the binary as data, never archive-controlled paths or symlinks.
tar -xOf "$staging/$asset" skillview > "$staging/skillview"
[[ -s "$staging/skillview" ]] || { echo "The release binary is empty." >&2; exit 1; }
mkdir -p "$bin_dir"
pending=$(mktemp "$bin_dir/.skillview.XXXXXX")
cp "$staging/skillview" "$pending"
chmod 755 "$pending"
if [[ "$force" == true ]]; then
  mv -f "$pending" "$target"
else
  # A hard link publishes without overwriting a command created during download.
  ln "$pending" "$target"
  rm -f "$pending"
fi
pending=""
echo "Installed $target ($version). No App, Go, or Node is required."
case ":$PATH:" in
  *":$bin_dir:"*) ;;
  *) printf 'Add this directory to PATH: %s\n' "$bin_dir" ;;
esac
echo "Verify with: skillview version"
