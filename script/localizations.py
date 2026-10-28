"""Generate macOS permission strings from the shared UI dictionary at build time."""
import json
import pathlib
import plistlib
import sys

ROOT = pathlib.Path(__file__).resolve().parents[1]

def generate(destination):
    languages = json.loads((ROOT / 'internal/locale/languages.json').read_text())
    messages = json.loads((ROOT / 'internal/locale/messages.json').read_text())
    info = plistlib.loads((ROOT / 'build/Info.plist').read_bytes())
    if info['CFBundleLocalizations'] != [language['mac'] for language in languages]:
        raise ValueError('Info.plist locales differ from languages.json')
    keys = [key for key in info if key.startswith('NS') and key.endswith('UsageDescription')]
    for language in languages:
        translations = messages[language['id']]
        lines = []
        for key in keys:
            source = info[key]
            value = translations[source]
            if not value.strip():
                raise ValueError(f'Empty permission translation: {language["id"]}/{key}')
            lines.append(f'{json.dumps(key)} = {json.dumps(value, ensure_ascii=False)};')
        folder = pathlib.Path(destination) / (language['mac'] + '.lproj')
        folder.mkdir(parents=True, exist_ok=True)
        (folder / 'InfoPlist.strings').write_text('\n'.join(lines) + '\n')

if __name__ == '__main__':
    if len(sys.argv) != 2:
        raise SystemExit('usage: localizations.py APP_RESOURCES_DIRECTORY')
    generate(sys.argv[1])
