# Research and reuse

Reviewed 2026-10-08. Paths are capability definitions, not proof that all Agents were tested on this Mac. The machine registry is `internal/library/locations.json`; changing a provider's locations requires updating that single file and the focused discovery tests.

## Agent locations

The [Vercel Skills Agent registry](https://github.com/vercel-labs/skills/blob/main/src/agents.ts) supplies the broad Agent definitions. Native additions and checks use official documentation:

Rechecked 2026-10-10 against [Skills CLI commit 13e4063](https://github.com/vercel-labs/skills/blob/13e4063a1cf913f5606d57d42ab83a86f5001e04/src/agents.ts). Its 79 named Agents are included in SkillView's 83 definitions. Project/global paths match the macOS registry, including fx's shared `.agents/skills` roots; legacy `.fx/skills` paths remain readable. This is directory compatibility, not live acceptance of every Agent or an online marketplace browser.

| Agent | Official reference | Locations covered |
| --- | --- | --- |
| Codex | [Skills](https://developers.openai.com/codex/skills) | `.agents/skills`, legacy `.codex/skills`, configured home, plugin caches and the older `$CODEX/.tmp/bundled-marketplace` plugin root |
| Claude Code | [Skills](https://code.claude.com/docs/en/skills) | `.claude/skills`, project Skills, installed plugin caches |
| Cursor | [Skills](https://prod.cursor.com/docs/skills) | `.cursor/skills`, project and user roots |
| GitHub Copilot | [Skills](https://docs.github.com/en/copilot/how-tos/copilot-on-github/customize-copilot/customize-cloud-agent/add-skills) | `.github/skills` and shared/project paths |
| VS Code | [Agent Skills](https://code.visualstudio.com/docs/agent-customization/agent-skills) | `.github/skills`, `.claude/skills`, `.agents/skills`; user `.copilot/skills`, `.claude/skills`, `.agents/skills` |
| Gemini CLI | [Agent Skills](https://geminicli.com/docs/cli/using-agent-skills/) | `.gemini/skills` and project paths |
| OpenCode | [Skills](https://dev.opencode.ai/v2/docs/skills/) | `.opencode/skills`, config roots, standalone root Markdown, shared paths |
| Cline | [Skills](https://docs.cline.bot/customization/skills) | `.cline/skills` and project paths |
| Windsurf / Devin | [Skills](https://docs.devin.ai/desktop/cascade/skills) | `.windsurf/skills`, `.devin/skills`, project and system paths |

Configuration and installation are separate: the app checks installed desktop apps and executables without launching Agents. Common Skills are shown as common files without claiming every Agent has executed them.

## Icons and UI

[Magpie](https://github.com/yetone/magpie) uses LobeHub brand assets and custom SVG interface controls. SkillView reuses [LobeHub Icons](https://github.com/lobehub/lobe-icons) for available brand logos and [Lucide](https://lucide.dev/) for interface controls and missing-brand fallbacks. Visual Studio Code uses the [Devicon v2.17.0 asset](https://github.com/devicons/devicon/blob/v2.17.0/icons/vscode/vscode-original.svg); Zed uses [Simple Icons](https://github.com/simple-icons/simple-icons/blob/develop/icons/zedindustries.svg). Their license notices are bundled as well. [React Virtuoso](https://virtuoso.dev/) provides list virtualization. Markdown rendering uses [react-markdown](https://github.com/remarkjs/react-markdown) and editing uses [CodeMirror](https://codemirror.net/).

## Providers and native capabilities

- [OpenAI Go SDK](https://github.com/openai/openai-go): configurable OpenAI-compatible translation endpoint, including local model servers.
- [Google Cloud Translation v2](https://docs.cloud.google.com/translate/docs/reference/rest/v2/translate): official keyed provider. The no-key Google public endpoint is an optional, unofficial alternative.
- [go-keychain](https://github.com/keybase/go-keychain): native macOS Keychain storage.
- [go-sqlite3](https://github.com/mattn/go-sqlite3): persistent, disposable caches.
- [fsnotify](https://github.com/fsnotify/fsnotify): filesystem notifications.
- [Goldmark](https://github.com/yuin/goldmark): Markdown parsing for translation boundaries and resource-link checks.
- [NSFileManager Trash](https://developer.apple.com/documentation/foundation/filemanager/trashitem(at:resultingitemurl:)): recoverable native file deletion.
- [Wails](https://v3.wails.io/): native window, dialogs, menu bindings and WebView lifecycle.

Pinned versions are in `go.mod` and `frontend/package-lock.json`. License texts are collected in `THIRD_PARTY_NOTICES.md` and included in the application bundle. No Magpie implementation code is copied into SkillView.
