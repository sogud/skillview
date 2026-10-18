# Archive Skills

Open a Skill and choose **Archive Skill**. Its entire folder moves to `~/.agents/.skillview-archive/`, outside the shared `skills/` directory. Archived Skills are excluded from scans. Other copies stay in place.

Archiving requires editing permission for the original folder in **Settings → Folders**. Linked entries remain read-only. The confirmation shows the selected copy, archive location and links that will lose their target.

Choose **Archived Skills → Restore** to move a folder back. Existing destinations are never overwritten. If the original folder is unavailable, open the archive in Finder.

Each item includes a `receipt.json` recovery record independent of app settings and SQLite. Interrupted moves and incomplete copies are retained for recovery. Nothing is purged automatically. Cross-volume moves preserve metadata and symbolic links.
