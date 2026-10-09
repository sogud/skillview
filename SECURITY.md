# Security policy

SkillView is experimental. Security fixes currently target the latest preview and `main`.

Use [GitHub's private vulnerability reporting](https://github.com/sogud/skillview/security/advisories/new) for a suspected security issue. Include the affected version, a minimal synthetic reproduction and expected behavior. Do not include API keys, account data, personal Skills or private filesystem contents in an issue or report. Ordinary bugs can be filed in public issues with disposable examples.

Skill content is data: the app never runs its instructions or scripts. Automatic and linked sources are read-only; editing is enabled only for an explicitly chosen original directory. Reads and writes check directory handles and revisions. Trash affects the selected original and preserves recovery information.

Markdown does not execute raw HTML or SVG, and remote images are not loaded automatically. Translation runs only after a user action and sends the selected document to the configured provider. Keys are stored in macOS Keychain. Derived catalog and translation caches remain local and can be cleared from Settings.

Preview packages have ad-hoc signatures and are not Apple-notarized. Gatekeeper, older macOS versions, native dependencies and unknown defects are outside the current acceptance; see [release readiness](RELEASE_READINESS.md).
