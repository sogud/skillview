# Plugins

Choose **Plugins** to browse installed Claude Code and Codex plugins by name, version and Agent. Open a plugin to read its files; README opens first when available.

Plugin files are read-only. Scripts and hooks are never executed. Resource links are not followed, and plugin content is not sent for translation.

## Discovery

- Claude Code: `$CLAUDE_CONFIG_DIR/plugins/installed_plugins.json`; defaults to `~/.claude`.
- Codex: `$CODEX_HOME/config.toml` and `plugins/cache`; defaults to `~/.codex`. Desktop installations need a valid `.codex-remote-plugin-install.json` marker. Disabled installations are included.

Unregistered caches and marketplace listings are excluded. Discovery runs when Plugins first opens or is refreshed.

A configured Codex version takes precedence; otherwise SkillView selects the highest semantic cache version, or the newest modified version folder. Each folder needs a manifest. This identifies browsable files, not the Agent’s active version or project enablement. See the [official cache layout](https://developers.openai.com/plugins/build/plugins#how-local-marketplaces-work).
