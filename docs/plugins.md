# Installed plugin browsing

Choose **Plugins** at the top of SkillView. The list shows plugin names, versions, descriptions and Agent logos. Search or filter by Claude Code or Codex, then open a plugin to browse its files. README files open first when present; Markdown, JSON, other ordinary text files and supported raster images can be previewed.

Back to Plugins has its own navigation row. Folder controls stay beside the file tree. Switching back to Skills preserves the Skill selection and filters, with the existing save/discard prompt protecting unsaved edits.

## Local installation records

- Claude Code: `$CLAUDE_CONFIG_DIR/plugins/installed_plugins.json`, falling back to `~/.claude`. The installed path and version come from each installation record.
- Codex: `$CODEX_HOME/config.toml` and `plugins/cache`, falling back to `~/.codex`. Configured plugins with an installed directory are included, including disabled installations. Desktop remote plugins are also included when their plugin folder has a valid `.codex-remote-plugin-install.json` installation marker.
- Unregistered and unmarked cache directories are excluded. Marketplace catalogs are not shown. Discovery reads installation metadata when Plugins is first opened or explicitly refreshed; it does not search Home or mounted volumes.

Codex cache layouts follow the [official plugin documentation](https://developers.openai.com/plugins/build/plugins#how-local-marketplaces-work). An explicit version in configuration takes precedence. Otherwise, SkillView browses the highest semantic version in the installed plugin's cache, or the most recently modified version folder when versions are not semantic. Version folders must contain a plugin manifest. This selects files for browsing; it does not establish the Agent's active runtime version or project-specific enablement.

## Read-only files

Plugin browsing reuses the bounded file reader and file tree. It includes dot-prefixed manifest folders, Skills, commands, agents, hooks and ordinary assets. Generated dependency folders such as `node_modules`, `.git` and virtual environments are omitted. Empty directories do not appear. Resource symlinks are displayed as disabled leaves and never followed. Removed or replaced plugin roots require refreshing the list.

The plugin browser does not install, update, enable, execute or edit plugins. There are no translation controls for plugin files. No plugin content is automatically sent to a provider.
