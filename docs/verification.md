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

## Unverified behavior

The ordinary app loaded a large real catalog with installed-Agent tabs; originals were not modified. This does not prove unrestricted access to every folder or execution of a Skill by an Agent. Permission failures and scan limits are reported.

The optional Google public endpoint succeeded once on synthetic text and failed in a later native check. AI and Google Cloud response/error behavior was checked with local fixtures, not paid accounts. Credential rollback used an in-memory key store; real Keychain acceptance remains outstanding.

Developer ID signing, Apple notarization, Gatekeeper installation on another Mac, the macOS 13 minimum and Intel hardware are not accepted. See [release readiness](../RELEASE_READINESS.md).
