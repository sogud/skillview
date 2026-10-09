"""Create disposable UI acceptance data, without reading real Skills."""
import json
from pathlib import Path
import sys

root = Path(sys.argv[1]).resolve()
root.mkdir(parents=True, exist_ok=False)
skills = {
    "code-review": "Review a focused change and explain what needs fixing.",
    "frontend-design": "Design clear, compact interfaces with deliberate visual choices.",
    "research": "Find primary sources and turn them into a useful answer.",
    "writing-for-agents": "Write instructions that are easy to follow and maintain.",
    "test-triage": "Read a failing test, identify the cause, and verify the repair.",
    "skill-creator": "Create a small Skill with a clear purpose and practical examples.",
}
for name, description in skills.items():
    folder = root / name
    (folder / "references").mkdir(parents=True)
    text = (
        f"---\n# Preserve this comment\nname: {name}\ndescription: {description}\n"
        "metadata:\n  fixture: true\n---\n\n"
        f"# {name}\n\n{description}\n\n"
        "## How to use\n\n"
        "1. Read the task and choose a small scope.\n"
        "2. Make the required change.\n"
        "3. Check the result and explain any remaining limits.\n\n"
        "Read the [reference guide](references/guide.md).\n\n"
        "```sh\nprintf 'example only'\n```\n"
    )
    (folder / "SKILL.md").write_text(text, encoding="utf-8")
    (folder / "references" / "guide.md").write_text(
        "# Reference guide\n\nThis is disposable acceptance data.\n", encoding="utf-8"
    )
    (folder / "sample.json").write_text(json.dumps({"example": True}), encoding="utf-8")
broken = root / "repair-me"
broken.mkdir()
(broken / "SKILL.md").write_text("---\nname: [broken\n---\n\nThis file can still be read and repaired.\n")
print(root)
