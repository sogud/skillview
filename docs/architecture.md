# Implementation notes

## Boundaries

`main.go` owns the native window, menus and lifecycle. `internal/app` exposes generated Wails bindings. `internal/library` owns discovery and file operations; `internal/translation` and `internal/cache` provide translation and SQLite caching. React retains the current selection and draft. The [original Magpie design reference](https://github.com/yetone/magpie) informed the compact list and reader; its example data is not a live catalog.

## Discovery and grouping

`internal/library/locations.json` is the Agent location registry. Automatic discovery reads known global locations and searches the user home and mounted local drives for project and custom `SKILL.md` files. It skips dependency trees, system caches, app bundles, Photos, backups and Trash. Discovery is limited to 250,000 directories and 64 levels; source catalog traversal has a separate 20,000-directory limit. Permission failures and truncation are reported. Selecting a narrower folder can cover an excluded or inaccessible location when the user has permission.

Scanning starts only through an explicit Scan action. Startup restores the saved catalog without folder traversal. Scan events report checked directories, discovered sources, completed source reads and Skill counts; cancellation retains the previous catalog. Source reads have a measured progress bar; discovery uses an indeterminate bar because the number of folders is not known in advance. The discovery walker resolves each root once and skips child symlinks and overlapping roots.

Shared roots appear once. Directory device/inode identities combine symlink aliases while retaining provenance. The UI groups physical entries by Skill name so one Skill occupies one row across Agents. A normalized content hash counts distinct versions without changing byte-exact file revisions. Each copy retains its path, resources, revision, favorite and write access. Reader copy selection is explicit; an Agent tab prefers a copy belonging to that Agent. Unknown brands use a generic icon, and Shared/Local are not counted as Agent platforms.

Installed detection requires a recognized desktop app, executable or a supported extension installation. Configuration-only entries remain visible in Settings when requested. Presence does not mean an Agent has actually executed a Skill.

## File handling

- Sources record resolved paths and filesystem identities. Access rechecks root links and opens components through held directory descriptors with `O_NOFOLLOW`. Intermediate absolute-path components use macOS search permission; the final directory must be readable.
- Automatic symlink targets can be read with cycle detection and captured identities. Linked-only entries remain read-only. Resource symlinks are listed and never followed.
- Only ordinary UTF-8 files up to 2 MiB are editable; hard links are read-only. Resource imports are bounded to 16 MiB. Raster previews are bounded; HTML and SVG never execute.
- Save checks a SHA-256 revision, creates a private Recovery copy, writes a sibling temporary file, preserves ACLs, extended attributes, ownership and mode, syncs, rechecks, and publishes through the held parent descriptor. Existing YAML and line endings are not reserialized.
- External changes keep the draft and expose original, disk and draft text for manual conflict resolution. Another process writing during the final rename cannot be controlled by a byte-revision check; avoid simultaneous saves from multiple apps.
- Creation and rename use `RENAME_EXCL`, never replacing same-name targets. Resource rename identifies affected local Markdown links and asks the user to update them; links are not rewritten automatically.
- Trash previews the exact selected original, count and known aliases, rechecks it, stages it within the same parent, and uses native `NSFileManager`. Failure restores the staged item rather than permanently deleting it. Other physical copies are untouched.
- Restore receipts persist original/Trash paths and filesystem identities. Restore rechecks the Trash item and original parent and refuses to replace an existing item. Finder remains available when the original parent was removed, the volume is missing or a cross-volume move cannot be completed.
- An explicitly writable containing folder can remove a final Skill link. The catalog captures its link revision, and removal rechecks the selected occurrence, current permission, parent and link identity. Resource links use the same Trash operation without opening their target. Receipts identify the link itself, allowing restart/undo to restore it while preserving its target.

Recovery copies are never pruned automatically. A source removal revokes catalog membership and persists an exclusion; it does not remove files.

## Translation, cache and UI

Google public translation uses an unofficial web endpoint; availability is not guaranteed. Google Cloud uses its documented API. AI uses `openai-go` with explicit settings, no ambient environment key, no tools, bounded responses, a timeout and cancellation. Documents are treated as text, not model instructions. Markdown text spans are translated while code, link destinations and structure remain intact. Model output is escaped before recombination; malformed or partial responses are rejected and not cached.

Google public translation uses GET requests. At most three redirects to the same scheme and host are followed when the HTTP method stays unchanged; AI POST redirects must preserve the method and body. Cross-origin, protocol-changing and method-changing redirects return a service-address error without forwarding content or credentials.

All providers share [go-retryablehttp](https://github.com/hashicorp/go-retryablehttp)'s request replay and cancellable waiting. Only HTTP 429 responses with a valid `Retry-After` of at most ten seconds receive up to two retries. Missing, invalid or longer hints return the rate-limit error without retrying early. Connection errors and uncertain model timeouts are not retried automatically; the AI SDK's separate retry loop is disabled. The complete HTTP request remains bounded to sixty seconds. Unmodified dependency source links and licenses are included in the bundle notices.

Keys live in the app's macOS Keychain service. Non-secret configuration is persisted separately. Changing settings rolls back key changes if configuration persistence fails. Derived translation cache keys include revision, provider, target, endpoint, model and parser/prompt version.

The reader occupies the main workspace, with a compact collapsible file tree on the left and the selected document on the right; Back to Skills retains list filters. Tree rows show basenames, retain complete paths as tooltips and support arrow-key navigation. Resource links are disabled leaves. Shared lightweight selection menus support keyboard selection and remain usable inside modal dialogs. Installed-Agent tabs wrap into rows without horizontal scrolling. Navigation and reading surfaces use neutral colors.

The reader's translation control also cancels an active request. Progress events use a request ID so canceled and old-document events cannot update the current view. The UI shows elapsed time, completed sections and a persistent success label only after the complete response and source revision check. AI uses batches of at most 24 sections or 12,000 characters (a single long section stays intact); completed batches are still rejected if a later batch fails. Switching documents or settings cancels and invalidates the old result. Incomplete or empty translated sections are rejected. Settings offers an explicit save-and-test action using a fixed `Hello.` sample without a cache hit. Copy includes a language label. Export uses a native save panel, rechecks the source revision and resolves the selected destination parent before opening it through held descriptors. A private sibling file is published with `RENAME_EXCL`; existing files and managed Skill directories are refused. Export makes no additional provider request.

SQLite caches catalog metadata, scanned file references and at most 500 complete translations. Cached references preserve directory identities; each read still checks current source permission, link targets and disk identity. Source bytes always come from disk. Old metadata-only caches require one manual scan before documents can reopen. Clearing derived caches preserves preferences, Recovery and Keychain.

The app does not run filesystem watchers or scan on window focus. Focus refreshes only the open document and its resources while keeping any draft. Explicit file/source operations refresh known catalog sources after the user's action. New external Skills appear after Scan. React Virtuoso renders visible rows rather than thousands of offscreen controls. Pointer clicks remove decorative focus boxes; keyboard navigation retains visible focus. Native menus, interface strings and settings support English and Simplified Chinese.

## Distribution

The bundle embeds the production frontend, its own icon and third-party notices. `SKILLVIEW_APP_BUNDLE` selects an isolated build location when needed. `SIGNING_IDENTITY` selects hardened-runtime Developer ID signing; absent that value the build is ad-hoc signed. `script/package.sh` verifies the bundle and creates ZIP/DMG packages with SHA-256 checksums. Packaging never uploads or publishes anything. The bundle and executable deployment target declare macOS 13; actual acceptance is on the current Apple Silicon Mac only. See the release readiness record before distribution.
