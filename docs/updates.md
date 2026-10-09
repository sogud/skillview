# Updates and publishing

## Using the app

The updater begins with alpha.2. Alpha.1 has no update checker, so install an updater-enabled build once from [GitHub Releases](https://github.com/sogud/skillview/releases).

SkillView checks at launch and every six hours without opening a dialog or downloading anything. When a newer compatible release is found, a purple download icon appears above Settings. Click it to read the release notes, download the update and restart. The app also has **Settings → General → Check for updates** and **SkillView → Check for Updates…**.

Before restarting, unsaved changes use the existing save/discard/cancel dialog. Cancelling keeps the current app and draft. Originals, settings and translation keys remain in their existing locations.

Preview builds receive preview and stable releases. Stable builds receive stable releases only. Versions follow semantic version ordering, including `alpha.2` → `alpha.3` → `0.1.0`; an equal or older version is not offered.

Install the app in `/Applications` or another writable directory on the same volume as macOS's temporary directory. The pinned Wails helper cannot rename an update across volumes; SkillView checks this before allowing in-app installation. Read-only, translocated and external-volume installations can use **View on GitHub** for a manual download. A failed download or verification never replaces the running app.

## Release authentication

The implementation reuses [Wails's updater](https://v3.wails.io/guides/updater/) and [Update Manifest protocol](https://v3.wails.io/reference/update-manifest/). GitHub selects the release and architecture-specific ZIP; its `manifest.json` provides the SHA-512 digest and Ed25519ph signature. SkillView pins the public key in `internal/updates/updater.key.pub` and refuses unsigned automatic installation. A checksum file alone is insufficient. Before restart, the staged bundle must have the expected app identifier, exact release version and a valid macOS code signature.

Update signing is separate from Apple's Developer ID signing and notarization. The current preview remains ad-hoc signed and not notarized. An update signature does not satisfy Gatekeeper's trust requirements for a first installation.

The private update key must remain outside the repository and build artifacts. Back it up securely; changing the pinned public key without an upgrade path prevents older apps from authenticating future releases. New contributors can build and test without possessing the release private key.

## Publishing a future version

1. Change `internal/updates/version.txt`, the release version's source of truth. For example, use `0.1.0-alpha.3` after alpha.2. The build derives the bundle metadata, package filenames and About version from it.
2. Run the focused updater tests and normal relevant checks. Build and review the actual ZIP, including its native UI and strict code signature. See [verification](verification.md) and [release readiness](../RELEASE_READINESS.md).
3. Sign the reviewed ZIP using the retained private key, then verify it against the pinned public key:

   ```sh
   export SKILLVIEW_UPDATE_PRIVATE_KEY='/path/outside/the/repository/updater.key'
   bash script/sign-update.sh \
     bin/releases/SkillView-0.1.0-alpha.3-macos-arm64.zip \
     /path/to/release-notes.md
   ```

   The script writes `manifest.json` alongside the ZIP. It does not publish anything or include the private key in the output. CI previews are unsigned update artifacts until the reviewed ZIP has been signed by the maintainer.

4. Commit and tag the reviewed source as `v0.1.0-alpha.3`. Publish a GitHub **prerelease** with the reviewed ZIP, its checksum, build information and **manifest.json** together. Use the corresponding stable tag and a regular release when stable acceptance is complete. Never reuse a published tag or replace its ZIP with different bytes.
5. Download the published assets independently and verify them. An already-installed previous version must discover the new release, download it, preserve/cancel unsaved edits correctly, replace the bundle and reopen with the new version. A package test alone does not confirm this complete installation path.

The built-in provider checks up to the ten most recent releases. Publishing a draft does not offer an update. A release without a manifest shows a manual download link; a malformed manifest or unsigned artifact is refused.

## Isolated acceptance

With `SKILLVIEW_SOURCE` set to a disposable Skill fixture, automatic update checks and installation are disabled. A manual check can use `SKILLVIEW_UPDATE_API=http://127.0.0.1:<port>` for a local synthetic GitHub API. Ordinary launches ignore that variable and always use the public repository. Never test bundle replacement against the user's normal app or real Skill folders.
