# Archive Skills for later

Open a Skill, then choose **Archive Skill** in the document toolbar. The confirmation shows the exact original copy, the archive folder and any known links that will lose their target. Only this copy moves; other copies with the same Skill name remain.

The default location is `~/.agents/.skillview-archive/`, beside `~/.agents/skills/`. Keeping it outside `skills/` avoids the standard shared Agent Skills discovery path. SkillView also excludes the archive and its temporary staging folders from its own scans, including explicitly added folders.

The archive contains the entire Skill folder: `SKILL.md`, references, scripts, assets and links. Nothing is executed or automatically deleted. Links inside a Skill stay links. A linked-only or read-only Skill cannot be archived; allow editing for its original directory in **Settings → Folders** first. This permission is separate from selecting a copy in a grouped list.

## Restore

Choose **Archived Skills** in the Skills list toolbar. Each entry shows its name, archive date and original location. **Restore** moves it back there. SkillView refuses to overwrite an existing file/folder or follow a replaced original parent. Both copies remain available if the original name is occupied. The folder button opens the archive in Finder for manual inspection when a location is unavailable.

Each archived item carries a `receipt.json` recovery record beside its payload, independently of SQLite and application preferences. Clearing the disposable app cache does not remove it. Archive and restore operations record recovery information before moving files; interrupted staging can be recovered from the archive list. Failed or incomplete copies are preserved for inspection. The app does not automatically purge or repair them.

Same-volume moves preserve filesystem identities through exclusive renames. Cross-volume moves use macOS Foundation's file mover and keep file metadata and symbolic links. Native acceptance uses isolated writable fixtures; it never archives real installed Skills automatically.
