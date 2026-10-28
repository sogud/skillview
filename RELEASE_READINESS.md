# Release readiness

The first regular GitHub release is `v0.1.0`; earlier builds are alpha prereleases. The source version is declared in [`internal/updates/version.txt`](internal/updates/version.txt). Packages carry ad-hoc signatures, not Developer ID signatures or Apple notarization. A regular version number does not imply Apple distribution acceptance or acceptance on every declared macOS version. The release provides a signed-update ZIP; the early alpha.1 DMG was excluded after a native mounting failure.

## Preview acceptance

- Local native acceptance covers Agent tabs and shared Skill rows, reading, editing with conflicts, creation, resource operations, Trash and restoration, language switching, translation copy/export/cancel and connection-test controls.
- Mutation and provider checks use disposable files, isolated settings and local HTTP fixtures. Real Skills were only read. Real configured AI/Google Cloud providers and real credential storage are not accepted integrations yet.
- Focused Go/race tests, frontend grouping tests, production frontend/native builds, local signature checks and portable ZIP/DMG checksums are part of the preview checks. The [macOS workflow](.github/workflows/preview.yml) performs these checks from an independent checkout; each Actions run is evidence for its own source commit.
- Source and history review is recorded in [open-source review](docs/open-source-review.md). Source licensing and bundled dependency notices are included.

See [verification](docs/verification.md) for the practical limits of the observed behavior. A successful build or checksum alone does not prove native launch or Apple notarization.

## Remaining platform and distribution acceptance

1. Sign with a real **Developer ID Application** identity, submit the concrete archive to Apple notarization, inspect acceptance, staple the app and recreate packages. Verify installation on another Mac with Gatekeeper enabled. Developer ID signing and notarization have not been performed for the published package.
2. Verify the declared macOS 13 minimum on real hardware, or raise the minimum to the oldest accepted OS. Local native acceptance is on macOS 27, Apple Silicon. Intel support requires a separate build and native acceptance.
3. Accept storage and retrieval through real macOS Keychain and configured AI/Google Cloud providers using authorized accounts. The no-key Google endpoint is unofficial: an earlier synthetic document succeeded, while a later native attempt reported a connection failure.
4. Review remaining Wails v3 and native-library risks, and refresh runtime dependency checks for each release. Automated scans cover known advisories, not every defect. Resource rename reports affected Markdown links and leaves their edits to the user.
5. A published 0.1.4 → 0.1.6 upgrade passed on an isolated App copy: signed download, unsaved-draft cancel, save, helper replacement, relaunch and settings preservation. This does not certify every old installation/environment or the discard branch. Keep a secure backup of the update signing key and validate each release's signature. See [updates and publishing](docs/updates.md).

## Packaging

```sh
bash script/package.sh
```

For an existing Developer ID identity, set `SIGNING_IDENTITY` when building. Signing uses hardened runtime and a timestamp. The scripts never create credentials, publish, disable Gatekeeper or remove quarantine attributes.

After a signing identity and notarization profile have been configured, an Apple-notarized package requires:

```sh
SIGNING_IDENTITY='Developer ID Application: YOUR EXISTING IDENTITY' bash script/package.sh
xcrun notarytool submit bin/releases/SkillView-0.1.0-macos-arm64.zip --keychain-profile YOUR_EXISTING_PROFILE --wait
xcrun stapler staple bin/SkillView.app
xcrun stapler validate bin/SkillView.app
spctl --assess --type execute --verbose=4 bin/SkillView.app
bash script/package.sh --skip-build
```

Notarization is a separate operation using the maintainer's existing Apple account; it has not been performed for the current release.
