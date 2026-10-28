# Verification

## Reader fixes (0.1.4)

Browser interaction with the production React components and synthetic service
responses verifies `./references/guide.md`, returning through `../SKILL.md`,
and refusal of an outside-root link. A 100-file tree has its own scroll area;
keyboard navigation reaches its last entry while the heading and controls stay
visible. The earlier completely unscrollable native symptom was not reproduced.

The same fixture verifies original/translated paragraphs, headings, nested list
items, quotations and table cells, with one unchanged code block. Parser checks
also cover reference links and complete-document fallback for incompatible block
structures. Thirteen path checks cover encoded names, traversal, absolute paths
and malformed input. Existing frontend checks and race-enabled library and
translation checks pass. Credentials and their storage are unchanged; no real
Skill or key was used for this acceptance.

The release binary scan identified GO-2026-6860 and GO-2026-6866 in
`golang.org/x/net` v0.60.0. The release updates it to the fixed v0.62.0, including
its required `x/sys` v0.49.0 and `x/text` v0.43.0 dependencies.

Native window inspection was interrupted by an unavailable computer-use
connection and timed out again after reconnection. The native binary built and
its isolated bundle passed strict signature verification, but native pointer
interaction and a complete in-app upgrade are not accepted for this release.

Local acceptance was performed on 2026-10-08–09 on an Apple Silicon Mac running macOS 27. Synthetic documents and isolated settings were used for mutation and provider checks. These checks support an experimental preview, not stable-release certification.

## Automated behavior checks

The focused Go tests cover safe file access, collisions, raw YAML/CRLF and file attributes, external-edit conflicts, directory replacement, linked/hard-linked files, discovery, shared Agent membership, independent copies, resources, Trash rollback and restoration after restart. Translation checks cover export without overwriting sources or aliases, cancellation, uncached fixed-sample connection tests, malicious output and bounded rate-limit retries. Cache and watcher behavior have their own package tests.

The three frontend grouping tests cover one description per Skill name, distinct versions and selection of each Agent's own copy without inheriting another copy's write access. TypeScript and the Vite/native production builds passed locally. Builds report large frontend/editor chunks and a duplicate Objective-C library warning.

ZIP/DMG checks cover checksums, archive integrity, the canonical `SkillView.app` path, executable equality, licenses and strict ad-hoc signature verification. The [Actions workflow](../.github/workflows/preview.yml) repeats the package tests, builds and runtime vulnerability checks on a separate macOS runner and records its source commit and tool versions. A successful Actions run applies only to that commit; runtime UI acceptance remains separate.

The production dependency Goldmark was updated from v1.7.16 to v1.7.17 after govulncheck identified [GO-2026-5320](https://pkg.go.dev/vuln/GO-2026-5320). The corrected binary passed the known-vulnerability scan locally. A fresh cloud scan then flagged twelve Go 1.27.1 standard-library advisories; the build was upgraded to the upstream security release, Go 1.27.2. Runtime checks exclude development dependencies, native libraries and unknown vulnerabilities. Each preview's CI run must refresh them before its packages are published.

## Native observations

- Search, selection, Markdown/resource reading, shortcuts, creation, saving, draft-preserving conflicts and quit confirmation were observed. Saved fixture bytes were inspected on disk.
- Automatic sources remained read-only; selecting a fixture source for editing enabled only that source. Favorites, theme and interface language persisted across restart. English/Chinese UI and native menus were observed. Pointer clicks removed the unwanted search focus box; keyboard focus remained visible.
- Same-name copies across Agents appeared in one row with overlapping logos/counts. Differing contents and each copy's access permissions remained available in the reader. Tab navigation, large-list scrolling and virtual rendering were observed.
- Resource creation/import/rename, external-change refresh, Trash and restoration were checked with real fixture files. Trashing one Skill copy preserved other Agent copies. Removing and restoring final Skill/resource symlinks preserved their targets, including restoration after restarting the app.
- Two native connection tests each sent only `Hello.` to a local OpenAI-compatible service. Translation retained inline code and link destinations. Copy included its target-language label. Export created a separate 0600 file; existing filenames and managed originals were preserved. Retrying after a collision succeeded with a new filename.
- Cancelling slow fixture translations returned to the original view. Late responses were neither displayed nor cached. Backend tests separately check HTTP context cancellation.

## Update checks (alpha.2 local build)

Focused updater package tests passed with race detection. They cover preview version ordering, platform-specific ZIP selection, missing manifests as manual-only downloads, malformed/unsigned manifests, mismatched release versions, corrupted downloads, wrong signing keys, cancellation, preview isolation and preservation of an already staged update.

The actual local alpha.2 ZIP passed Ed25519ph verification with the pinned release public key, Wails download/extraction, and checks of the staged app identifier, exact release version and strict macOS code signature. Its archive contains one canonical `SkillView.app`, with no private signing key or `__MACOSX` top-level directory. A separate disposable native helper check replaced an old stub app, cleaned its backup/staging directory and produced a relaunch receipt. The user's installed app was not replaced.

Native observations used an isolated local release API: the purple sidebar badge appeared for alpha.3, Settings and the native menu opened the release notes, checking the current version cleared the badge, network failure exposed retry/GitHub controls, and Chinese UI labels were observed. Preview fixtures disabled automatic checking and installation. This is not acceptance of the full save/discard/restart flow against a published future version.

The newly used `golang.org/x/mod` dependency was upgraded to v0.40.0 after the binary scan identified [GO-2026-6180](https://pkg.go.dev/vuln/GO-2026-6180) and [GO-2026-6179](https://pkg.go.dev/vuln/GO-2026-6179). The corrected alpha.2 binary passed the known-vulnerability scan. See [update publishing](updates.md) for the signing and installation requirements.

## Feedback fixes (alpha.3)

Startup tests confirm that known Skills are readable before the wider search starts. Discovery/reading progress tests cover real counters, overlapping and linked roots and cancellation without a false completion. Native observations used 80 synthetic Skills and 3,000 empty fixture directories; installed-Agent tabs wrapped into rows, reading occupied the main area, Back to Skills restored the list and navigation backgrounds were neutral.

A local OpenAI-compatible fixture showed completed-section progress, elapsed time, the translated document with a persistent completion label, cancellation returning to the original and localized HTTP 503 feedback with retry/configuration controls. AI batch tests reject incomplete and partially failed documents without caching them. No real Skill was sent to a translation provider. The public Google endpoint translated the fixed `Hello.` connection sample successfully on this Mac; availability on another computer remains unverified.

A public GitHub API check reproduced rate limiting. The fallback through the public Atom release feed then returned the correct up-to-date state. Fixture tests verify that the fallback still rejects unsigned manifests and tampered ZIPs, preserves stable/preview channel filtering and never offers an older version. Static macOS HTTP/HTTPS proxy support uses [go-ieproxy](https://github.com/mattn/go-ieproxy); loopback models are tested to bypass proxies. PAC and SOCKS-only configurations are not evaluated.

The new dependency's initial `golang.org/x/net` v0.58.0 was flagged by the binary scan for five known HTTP/2 advisories, including [GO-2026-6610](https://pkg.go.dev/vuln/GO-2026-6610). It was upgraded to the listed fixed version v0.60.0 before packaging. Package checks and CI evidence apply to the exact reviewed source and ZIP.

## Manual scanning and file tree (alpha.4)

Focused cache tests cover reopening without traversal, reading current disk bytes, replaced folders, removed sources, current automatic-link targets and metadata-only cache migration. Redirect tests cover Google GET 302, AI POST 307 with unchanged body, bounded redirect loops and refusal to forward requests to another origin. Five frontend tests cover file-tree nesting/collapse and shared Skill grouping. TypeScript, production builds and focused vet checks passed.

Native acceptance used an isolated synthetic Skill with two copies and nested references, scripts and assets. Fresh startup waited for Scan; restart opened the cached catalog without scanning. An externally added Skill stayed absent until Scan was clicked. Tree expansion, file selection, resource reading, copy selection with keyboard controls and unsaved-draft protection were observed; discarded fixture edits left original bytes unchanged. Settings language/provider menus opened inside the modal and retained the selected checkmark.

A local AI fixture redirected with HTTP 307, received the same request body and showed translation progress followed by translated text and the completion label. The public Google fixed `Hello.` sample returned HTTP 429 on this Mac during this check. Same-origin 302 handling passed locally, but Google availability on the user's other computer remains unverified. No real Skill was sent to a provider.

## Known Agent scans, appearance and diagnostics (alpha.5)

Focused race-enabled checks cover retiring old automatic sources outside the registry, keeping explicitly added folders, discovering Codex's legacy bundled-plugin root, folder-reading progress and theme-color persistence/validation. The updater tests reproduce a connection closed halfway and a download that outlives the short release-check timeout. They verify resumed bytes through Wails's original full-archive signature and extraction path, reject ignored/wrong ranges and tampered resumed archives, and stop retries on cancellation. Diagnostics tests confirm persistence, rotation, private permissions and exclusion of credential-bearing URLs and local paths.

An isolated native preview showed only library and Settings navigation, manual scanning, the theme-color palette and readable update diagnostics. The selected color survived a real app restart, which reopened the cached catalog without scanning. An unreachable loopback update service produced localized retry controls and a persisted, classified connection failure in the diagnostics view. No real Skill contents were transmitted.

## Free translation services and navigation (alpha.6)

Focused translation/app tests passed with race detection. They cover batched Google and Youdao requests, request pacing and per-provider cooldown, anonymous Youdao cookies/signatures/encrypted responses, malformed or incomplete responses, cancellation, caching, LibreTranslate batches and independent credential rollback. Thirty short Google spans use two requests instead of thirty. Incomplete translations are neither shown as complete nor cached.

The production translation service successfully translated a fixed synthetic two-section Markdown sample through both public Google and anonymous Youdao endpoints on this Mac. No real Skill was transmitted. LibreTranslate used a local synthetic API fixture; no public hosted instance or paid AI account was tested.

A browser fixture rendered the actual React components with isolated API responses. It showed no left navigation rail, Settings and update controls at the top right, and exactly four translation choices: Google, Youdao, self-hosted LibreTranslate and AI. Selecting Youdao showed no key field; saving and connection-test completion labels were observed with mock responses. The final TypeScript/frontend and native production builds passed. Native window clicks were not checked for this version.

## Folder settings and long paragraphs (alpha.7)

Focused race-enabled translation/app checks cover sentence boundaries, Unicode/emoji limits, large documents above the previous 128 KiB cap, the 1 MiB maximum, per-chunk progress and failure/cancellation without partial results. A 2,320-character paragraph produced three bounded requests and progress from 0/3 through 3/3 while preserving adjacent inline code. Fixed synthetic samples of 900, 1,500, 3,000 and 6,000 characters translated successfully through the public Youdao service; the user's exact red error was not reproduced.

A browser fixture showed the additional-folder explanation, a 26 px Add button and no empty list occupying space. Adding read-only/editable folders passed the selected permission to the mock picker; removing a mock folder restored the empty state. These browser checks do not verify the native macOS folder picker. No real folders or Skills were modified or sent to a service.

## Navigation and first regular release (0.1.0)

Back to Skills is the only action in a separate navigation row. File-tree controls and current-document actions belong to their own panes. An isolated native macOS build with disposable documents verified collapsing folders, hiding the tree without leaving the reader, Enter to restore it with focus on the selected file, reference reading and returning to the list. Original fixture bytes remained unchanged. This does not exercise Apple Gatekeeper installation or live providers.

Focused update tests verify stable/alpha ordering, stable feed filtering and signed download/extraction; file-tree unit checks pass. TypeScript and the native production build pass, including strict ad-hoc signature verification. A production React browser fixture verifies separated control positions, keyboard focus, and no horizontal page overflow at 640 px. The tested reader and the introduction website have zero automated accessibility violations. Website checks cover English/Chinese switching, loaded demo imagery and a 390 px viewport without horizontal overflow. The screenshot uses only synthetic content. No real Skills or private account data are included in the website.

## Installed plugin browsing (2026-10-14)

Focused race-enabled library tests verify Claude Code installation records, disabled Codex installations, desktop remote installation markers, semantic and pinned cache versions, exclusion of unregistered caches and staging folders, malformed metadata without private parse-error details, missing pinned versions, replaced roots, symlink refusal and rejection of plugin identifiers by Skill write APIs. Existing resource and manual-scan behavior is checked separately.

Production React browser fixtures verify first-open metadata loading, list search, Agent filters, empty/error states, version/disabled labels, relative Markdown file links, read-only controls and retained Skill selection across mode switches. The tested Chinese/light and English/dark plugin lists and readers have zero automated accessibility violations after transitions settle; the 640 px layout has no horizontal page overflow.

An isolated native Apple Silicon build with two synthetic installations verifies the Skills/Plugins switch, Claude Code and Codex names/versions, folder expansion, README reading, relative document navigation, JSON manifest preview and Back to Plugins. The production build and strict ad-hoc signature verification pass. Fixtures and generated screenshots remain ignored; no installed plugins are edited or executed. Other machines, project-specific enablement and the Agent's exact active runtime version are not verified.

## Skill archiving

Focused race-enabled tests cover complete-folder archive/restore, recovery after replacing application settings, preservation of independent copies and resource symlinks, exclusion from regular and explicitly selected archive sources, stale previews, read-only sources, failed moves and staged interruption recovery. Restoration refuses occupied destinations, replaced payloads/parents and linked archive roots. A disposable fixture crossing the external SSD and system volume verifies original bytes, file permissions and relative links after archive and restore.

The production frontend and native build pass, with strict ad-hoc signature verification. Browser fixtures exercise archive confirmation, removal from the active list, the archive list, a collision with both copies retained, successful restoration, the empty archive and an empty active list with a restore entry. Chinese/light and English/dark dialogs have zero settled automated accessibility violations; the tested 640 px viewport has no horizontal page or dialog overflow. These fixtures do not modify real Skills. Native pointer acceptance subsequently passed with the shared CLI fixture below.

## Shared CLI and Skill

Focused race-enabled tests cover shared settings and write locks, revoked editing access, stale previews, sync boundaries, recoverable cleanup, resource backups, security-rule matching, current-Agent filtering and command installation conflicts. TypeScript, the production native build, focused vet checks and strict ad-hoc signature verification pass.

An isolated native fixture verifies manual scanning, reading, archive/restore and command installation. The installed CLI reads the same catalog, cold-starts the app and activates one running window. CLI archive/restore updates the app's list on activation without scanning; restored bytes match the original. Doctor, topology, graph, backup and security commands run against synthetic files. No real Skills are changed or sent to translation services.

The local skill is now `skillview`, backed by this repository's CLI. Downloading or installing Skills is not supported. Distribution signing and another-machine installation remain unverified.

## Remaining acceptance

Alpha.9 checks compare all 79 named Agents and their macOS project/global paths in the pinned Vercel Skills CLI registry with SkillView's 83 definitions. Focused discovery tests verify fx membership in the shared root and preservation of legacy roots. A browser fixture verifies that hiding, restoring and collapsing the tree keep the reader open, Back returns to the list, and the tree visibility controls are outside the top navigation. English/Chinese labels and the tested light/open and dark/hidden reader states pass with zero automated accessibility violations. SQLite persistence and translation cache/failure tests pass with race detection; no live provider requests are made. Frontend and native production builds pass with strict ad-hoc signature verification. Native window clicks are not exercised in this pass.

Alpha.8 reading-layout checks use isolated preferences to verify old settings receive 14 px/244 px defaults, changes survive reload, invalid ranges are rejected and a failed save preserves previous values. A browser fixture exercises the production UI with synthetic documents and a mock service: drag and keyboard resizing, restored width/font size, the 20 px limit, 640 px window layout, wide-table containment, keyboard scrolling and translation progress/completion/failure states. The reader has zero automated accessibility violations in the tested final light/dark states. Translation providers and native macOS interactions are not exercised by this fixture.

The ordinary app loaded a large real catalog with installed-Agent tabs; originals were not modified. This does not prove unrestricted access to every folder or execution of a Skill by an Agent. Permission failures and scan limits are reported.

The optional Google public endpoint succeeded once on synthetic text and failed in a later native check. AI and Google Cloud response/error behavior was checked with local fixtures, not paid accounts. Credential rollback used an in-memory key store; real Keychain acceptance remains outstanding.

Developer ID signing, Apple notarization, Gatekeeper installation on another Mac, the macOS 13 minimum and Intel hardware are not accepted. See [release readiness](../RELEASE_READINESS.md).
