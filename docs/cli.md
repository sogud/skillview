# Command line

Choose **Settings → General → Install command line**, then add `~/.local/bin` to PATH if needed. The command uses the installed app's executable and updates with it. Reinstall the command after moving the app; existing commands are never overwritten.

```sh
skillview app
skillview scan
skillview list --json
skillview list --current --json
skillview show <id-or-path>
skillview help
```

App and CLI share the Agent registry, settings, catalog, permissions and archive. `list` uses the saved catalog; `scan` refreshes it. `app` opens the main window only. No Skill downloads or installation.

Archive, sync and cleanup show a preview first. Apply it with `--apply --token <token>`. Restore uses `skillview restore <archive-id> --apply`. To approve an original folder explicitly, use `skillview folders add <path> --writable --apply`. Other folders and linked Skills stay read-only.

Sync shares existing Skills through links; originals stay in place. Defaults cover Agents and Claude. Explicitly selected private platforms receive shared links without importing their own Skills back. Conflicts are reported and preserved. Cleanup moves only broken links and truly empty directories to macOS Trash.

Backups are private tar.gz files under the app data directory's `backups/`, including a manifest and resource links. Security checks read local text and report rule locations without printing matched secrets. Recommendations use names and descriptions in the selected/current Agent scope.

Commands offer `--json`; progress and failures go to stderr. File mutations and scans share a process lock. When the app regains focus, it reloads the catalog without discarding drafts or rescanning.
