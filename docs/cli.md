# Command line

## Install without the app

The standalone CLI supports macOS Apple Silicon. It does not require the App, Go or Node. After a release includes CLI assets, download and inspect its installer, then run it:

```sh
curl --fail --location --proto '=https' --proto-redir '=https' \
  https://github.com/sogud/skillview/releases/latest/download/install-cli.sh \
  --output install-skillview-cli.sh
# Read install-skillview-cli.sh before running it.
bash install-skillview-cli.sh --dry-run
bash install-skillview-cli.sh
```

From a source checkout, use `bash script/install-cli.sh` instead.

The installer resolves the latest stable GitHub release, downloads its `skillview-cli-<version>-macos-arm64.tar.gz` and SHA-256 file, verifies the archive, and installs a regular executable at `~/.local/bin/skillview`. The checksum detects corrupted downloads; it is not a separate publisher signature. The installer never runs the downloaded executable or modifies shell settings.

**Release availability:** app-only releases, including v0.1.3, do not contain standalone CLI assets. The release installer becomes usable after those assets are published. Until then, build from the checkout with `bash script/build-cli.sh`, then run `bin/skillview` directly or `bin/skillview cli install`. Building requires Go and Xcode Command Line Tools, not the frontend or Wails tools.

Use `--version <tag>` to pin a release or `--bin-dir <path>` to choose another destination. Existing commands, including links installed by the App, are preserved. Inspect them before using `--force` to replace the selected entry. Checksum or download failure leaves the existing command unchanged. Re-run with `--force` to update an existing standalone installation after reviewing the release.

Add `~/.local/bin` to PATH if needed, then run `skillview version` and `skillview help`. For the current terminal: `export PATH="$HOME/.local/bin:$PATH"`. Add that line once to the appropriate shell startup file for future terminals.

## Optional app

`skillview app` opens an installed SkillView App. If none is installed, it reports that the App is required only for this command. Other commands work independently.

The App can still install a linked CLI through **Settings → General → Install command line**. That entry uses the App's executable and updates with it; moving the App requires relinking. Existing commands are never overwritten.

## Usage

```sh
skillview app
skillview scan
skillview list --json
skillview list --current --json
skillview show <id-or-path>
skillview help
```

The standalone CLI and optional App share the Agent registry, settings, catalog, permissions and archive. `list` uses the saved catalog; `scan` refreshes it. `app` opens the main window only. No Skill downloads or installation.

Archive, sync and cleanup show a preview first. Apply it with `--apply --token <token>`. Restore uses `skillview restore <archive-id> --apply`. To approve an original folder explicitly, use `skillview folders add <path> --writable --apply`. Other folders and linked Skills stay read-only.

Sync shares existing Skills through links; originals stay in place. Defaults cover Agents and Claude. Explicitly selected private platforms receive shared links without importing their own Skills back. Conflicts are reported and preserved. Cleanup moves only broken links and truly empty directories to macOS Trash.

Backups are private tar.gz files under the app data directory's `backups/`, including a manifest and resource links. Security checks read local text and report rule locations without printing matched secrets. Recommendations use names and descriptions in the selected/current Agent scope.

Commands offer `--json`; progress and failures go to stderr. File mutations and scans share a process lock. When the app regains focus, it reloads the catalog without discarding drafts or rescanning.
