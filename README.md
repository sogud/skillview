# SkillView

Read, translate and manage local Agent Skills and plugins on macOS.

![SkillView with example Skills](docs/images/skillview-demo.jpg)

[Download](https://github.com/sogud/skillview/releases/latest) · [Website](https://sogud.github.io/skillview/)

Apple Silicon. macOS 13+ declared; the minimum version remains unverified. Ad-hoc signed, not Apple notarized. If macOS blocks the app, see [Apple’s guidance](https://support.apple.com/en-us/102445).

## Features

- Browse installed Agents’ Skills. Shared copies appear once with their Agent logos.
- Read Markdown and resources in a resizable file tree. Scan manually; reopen the saved catalog at startup.
- Browse installed Claude Code and Codex [plugins](docs/plugins.md).
- Translate with Google, Youdao, self-hosted LibreTranslate or compatible AI. Completed translations are cached.
- [Archive and restore Skills](docs/archive.md). Archived folders stay in `~/.agents/.skillview-archive/`.
- English/Chinese interface, light/dark themes, accent color and reading size.
- Signed downloads and [in-app updates](docs/updates.md).

Folders are read-only until you allow editing in Settings. Changes apply to the selected original copy. Skill scripts and plugin hooks are never executed. Translation sends the selected document to your chosen provider only when requested; API keys stay in macOS Keychain. No telemetry or cloud sync.

## Build

Requires an Apple Silicon Mac, Node/npm and Xcode Command Line Tools. The bootstrap installs pinned Go and Wails tools when needed.

```sh
git clone https://github.com/sogud/skillview.git
cd skillview
bash script/build_and_run.sh
```

Build only: `bash script/build.sh`. Package: `bash script/package.sh`.

Tools use `runtime/`, temporary files use `tmp/`, and bundles use `bin/`. Override with `SKILLVIEW_TMP_DIR`, `SKILLVIEW_APP_BUNDLE` and `SKILLVIEW_RELEASE_DIR`. App settings and caches use `~/Library/Application Support/SkillView`.

Checks:

```sh
source script/toolchain.sh
ensure_go
go test -race ./internal/library ./internal/translation ./internal/cache ./internal/watch ./internal/updates
npm --prefix frontend ci
npm --prefix frontend run build
node --test frontend/src/skill-groups.test.mjs frontend/src/file-tree.test.mjs
```

The static website lives in `website/`; GitHub Pages deploys after a release. Preview with `python3 -m http.server 9380 --directory website`.

## Documentation

[Translation](docs/translation.md) · [Architecture](docs/architecture.md) · [Agent sources](docs/sources.md) · [Verification](docs/verification.md) · [Release readiness](RELEASE_READINESS.md) · [Open-source review](docs/open-source-review.md) · [Contributing](CONTRIBUTING.md) · [Security](SECURITY.md)

## License

[MIT](LICENSE). Dependency and asset licenses are in [third-party notices](THIRD_PARTY_NOTICES.md). Agent logos identify compatible tools. [Magpie](https://github.com/yetone/magpie) inspired the interface; [Vercel Skills](https://github.com/vercel-labs/skills) supplied the Agent registry. Screenshots use example content.
