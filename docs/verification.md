# Verification

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

## Remaining acceptance

The ordinary app loaded a large real catalog with installed-Agent tabs; originals were not modified. This does not prove unrestricted access to every folder or execution of a Skill by an Agent. Permission failures and scan limits are reported.

The optional Google public endpoint succeeded once on synthetic text and failed in a later native check. AI and Google Cloud response/error behavior was checked with local fixtures, not paid accounts. Credential rollback used an in-memory key store; real Keychain acceptance remains outstanding.

Developer ID signing, Apple notarization, Gatekeeper installation on another Mac, the macOS 13 minimum and Intel hardware are not accepted. See [release readiness](../RELEASE_READINESS.md).
