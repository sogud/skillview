# Interface languages

SkillView includes 18 interface languages: English (`en`), 简体中文 (`zh-CN`), 繁體中文 (`zh-TW`), 日本語 (`ja`), 한국어 (`ko`), Français (`fr`), Deutsch (`de`), Español (`es`), Português (Brasil) (`pt-BR`), Italiano (`it`), Русский (`ru`), العربية (`ar`), हिन्दी (`hi`), Bahasa Indonesia (`id`), Tiếng Việt (`vi`), ไทย (`th`), Türkçe (`tr`) and Nederlands (`nl`). Choose a language in Settings or follow the system. This setting is separate from the document translation target.

`internal/locale/messages.json` is the shared Go/TypeScript dictionary. All 18 dictionaries contain 490 keys, including feedback diagnostics and macOS permission explanations. English sentences remain keys and fallback text. `languages.json` defines names, direction and macOS identifiers. No translation service is contacted to display the interface. x18n is not installed or required.

The 16 newly added languages were translated by a model from public UI strings and have not been reviewed by native speakers. Product names and code identifiers are preserved. Some conventional technical terms (including the Agents section name) remain in English. Formatting uses simple named placeholders and locale-aware numbers; it is not an ICU message engine and does not implement grammatical plural categories. Count-neutral wording is used in some translations; singular/plural phrasing remains an editorial review item.

## Resolution and direction

The first supported system preference wins. Regional variants use the supported base language; Portuguese variants use Brazilian Portuguese. Chinese `Hant` or regions TW/HK/MO use Traditional Chinese; `Hans` takes precedence over region and uses Simplified Chinese. Unknown languages and missing keys fall back to English. Preferences are persisted as canonical supported IDs or `system`.

Arabic switches the interface to RTL. Code, paths, file trees and URL/key fields remain LTR; substituted values are isolated within RTL labels. Markdown and document titles choose direction from their content. File-tree resizing follows interface direction.

Go embeds the same dictionary for native menus and dialogs. `script/localizations.py` generates the 18 macOS `InfoPlist.strings` resources from it during packaging; `build/Info.plist` declares their language codes. The old English/Chinese source `.lproj` files are retained, while the build now uses the shared dictionary.

## Validation recorded on 2026-10-28

- Exact key coverage, nonempty values and placeholder multiplicity passed for all 18 dictionaries. Newly translated dictionaries were also checked for accidental bidi controls and unchanged long English text; shared words, names and pure placeholder formats are intentional.
- Go and TypeScript use the same 30 matching fixtures. Checks include Chinese script/region precedence, regional variants, preferred-language order, English fallback and RTL interpolation. Go also reads the current macOS language list without changing it.
- A disposable library test persisted and reopened all 18 language preferences, plus system mode, alias normalization and invalid-input rejection.
- Go locale/app checks, focused library checks, generated Wails bindings, TypeScript/Vite build and diff whitespace checks passed. The existing duplicate `-lobjc` linker and large JavaScript chunk warnings remain; the added dictionaries increase the main bundle.
- All 18 generated macOS language resource files passed `plutil -lint` in disposable build output.
- Actual browser UI acceptance used the real frontend with an isolated synthetic Wails bridge: Chinese settings, all 18 native-name options, German long translation-provider help, Arabic settings/reader, code and secondary paths remaining LTR, and RTL keyboard resizing (244 to 260 with Left). System `zh-HK` selected Traditional Chinese after reload; unsupported `sv-SE` selected English. Inspected Chinese/German/Arabic pages had no horizontal page overflow. The synthetic bridge never invokes real scanning, translation, feedback or filesystem operations.

During live source updates, Vite Fast Refresh temporarily reported `LocaleProvider is missing`; a fresh full-page load and settings interaction produced no console errors. The temporary browser tabs and loopback preview process were closed after acceptance.

The UI inspection is not a native packaged-app acceptance test. Native macOS menu rendering, OS permission prompts, every screen in all 18 languages, all window sizes and native-speaker translation review remain unverified. No app was installed, replaced or released, and no user documents or credentials were used. Feedback service deployment remains separately pending approval.

Focused checks:

```sh
node --test frontend/src/locales.test.mjs
go test ./internal/locale
go test ./internal/library -run '^(TestInterfaceLanguagePersistsAcrossReopen|TestSharedWritersReloadSettingsAndRefuseRevokedAccess)$'
go test ./internal/app
npm --prefix frontend run build
```
