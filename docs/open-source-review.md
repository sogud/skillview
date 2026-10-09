# Open-source review

Reviewed on 2026-10-09 for the first experimental preview. This review covers the standalone SkillView source, its published Git history, dependency notices, build workflow and preview package contents. Private parent-workspace history, local settings, acceptance data and generated caches are excluded from publication.

## Findings addressed before publication

- **Missing project license:** added an MIT license for SkillView and included it in the application bundle. Third-party code and assets keep their existing licenses.
- **Private-workspace assumptions:** replaced parent-only build instructions and workflow paths with standalone commands and CI. Local Codex configuration, generated files, secrets and acceptance artifacts are ignored.
- **Internal acceptance material:** public documentation describes synthetic checks and their limits without publishing personal catalogs, machine paths, private screenshots, session records or recovery data.
- **Dependency advisory:** the pinned Goldmark version is v1.7.17, fixing the earlier known renderer advisory. Runtime Go and npm scans are repeated by preview CI.
- **Go runtime advisories:** the first cloud binary scan flagged twelve standard-library advisories in Go 1.27.1. The build now pins [Go 1.27.2](https://go.dev/doc/devel/release#go1.27.0), the upstream security fix, and upgrades older local bootstrap installations before compiling.

## Scope and evidence

Gitleaks v8.30.1 scanned the selected source without finding credentials. Manual review checked personal filesystem paths, credentials/configuration, registry provenance, third-party attribution and build/package boundaries. The project had no existing tracked files or commits in its private parent repository; its new independent history is reviewed in full before the first push. Commit metadata uses a GitHub noreply address.

The application includes collected runtime dependency license texts plus notices for the Vercel Agent registry, LobeHub, Devicon and Simple Icons. The two MPL-2.0 HashiCorp libraries are unmodified; notices provide their exact upstream source URLs. Brand icons identify tools and do not imply affiliation. Magpie informed the visual direction; its implementation was not copied.

## Alpha.2 update review

The updater change was reviewed before the second experimental release. Gitleaks and a separate path/key check covered the 115 selected public source files; local settings, update fixtures, builds and the private signing key were excluded. The only committed update key is the public verification key. Complete standalone Git history and commit metadata were checked again before pushing.

The new runtime dependency `golang.org/x/mod` is pinned to v0.40.0 after two known-advisory findings; its BSD license is included in the regenerated notices. The final binary's vulnerability scan passed. The reviewed ZIP contains one app bundle and no private key, local settings or acceptance files. Its digest and Ed25519ph signature were verified against the pinned public key.

No disclosure blocker was found. The release remains experimental: Apple signing/notarization and the complete published-version upgrade flow have the acceptance limits described in [verification](verification.md) and [release readiness](../RELEASE_READINESS.md).

## Alpha.3 feedback review

The selected 126 public source files and complete three-commit standalone history passed Gitleaks before the feedback fix commit. Manual identity/path checks found no private machine paths or personal accounts in the new files. Upstream authors' required license contacts remain in dependency notices. User-provided screenshots, native acceptance screenshots, catalogs, temporary provider servers, logs and signing credentials remain excluded.

The system-proxy helper is the unmodified MIT-licensed `go-ieproxy` v0.0.12. Notices cover it and its transitive dependencies. The binary scan flagged five known HTTP/2 advisories in its initial `golang.org/x/net` v0.58.0 dependency; the dependency was upgraded to v0.60.0 and the rebuilt binary passed. The frontend runtime audit also passed. The updater's feed fallback retains the pinned public key and signed-ZIP verification; tests reject unsigned and tampered fallback downloads.

No disclosure blocker was found for this change. Real provider accounts, PAC/SOCKS proxy configurations and published-version replacement retain the acceptance limits below.

## Alpha.4 manual scanning and file tree review

The selected 133 public files and complete four-commit standalone history passed Gitleaks before the new commit. Private-path, identity, internal-address, historical filename and commit-metadata checks found no disclosure blocker. Synthetic catalogs, local provider servers, logs, screenshots and the private update-signing key remain excluded. The only update key in source is the public verification key.

The change introduces no runtime dependency. Regenerated notices omit the filesystem-watcher dependency that is no longer linked into the app. File-tree actions reuse the existing revision, permission and Trash checks. Cached references recheck current source access, directory identities and automatic-link targets before opening a file. Redirect tests confirm that translation content and credentials are not forwarded to another origin; same-origin AI redirects preserve the POST body.

The MIT license, contribution/security documents, English README, Chinese interface coverage and pinned CI remain present. A separate code of conduct remains optional governance work. CI runtime scans and review of the exact downloadable ZIP must pass before this preview is published. This review does not accept paid translation accounts, Google availability on another computer, Apple notarization or the complete published-version upgrade flow.

## Alpha.5 scope

This change introduces no dependency. Automatic scanning is restricted to registered Agent locations; previous broad automatic sources lose catalog membership without deleting their files. Explicit folders and write approvals remain. Favorites and source-navigation controls are removed. Theme colors are validated before persistence. Interrupted update requests append only an exact matching byte range and retain SHA-512/Ed25519ph verification over the entire archive. Update diagnostics store fixed stages, versions, byte counts and classified errors locally; SDK helper paths are not copied into the report.

Synthetic tests, UI screenshots, local logs, update acceptance data and private signing credentials remain excluded from the public source and packages. Gitleaks scanned the selected 137 public source files and complete six-commit history without finding credentials. Manual checks found no private machine paths, personal commit email or sensitive historical filename. Review of the exact downloadable ZIP and fresh CI runtime scans remain required before publication. Apple notarization and acceptance on the user's other Mac remain outside this local review.

## Coverage limits

This is a source-disclosure and license review, not a guarantee that the app has no vulnerabilities. Secret scanning can miss unknown formats. Live API accounts, real Keychain behavior, development/native dependency advisories, older macOS hardware, Intel builds and Apple notarization are not covered. Preview packages are explicitly labeled ad-hoc signed and unnotarized. See [security](../SECURITY.md), [verification](verification.md) and [release readiness](../RELEASE_READINESS.md).
