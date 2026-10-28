import languageData from '../../internal/locale/languages.json' with { type: 'json' }
import messageData from '../../internal/locale/messages.json' with { type: 'json' }

export const languages = languageData
const dictionaries = messageData as Record<string, Record<string, string>>

export function matchLocale(tag: string): string | undefined {
  const parts = tag
    .trim()
    .replaceAll('_', '-')
    .toLowerCase()
    .split(/[.@]/)[0]
    .split('-')
  let candidate = parts[0]
  if (candidate === 'zh') {
    candidate =
      parts.includes('hant') ||
      (!parts.includes('hans') &&
        parts.some((part) => ['tw', 'hk', 'mo'].includes(part)))
        ? 'zh-TW'
        : 'zh-CN'
  } else if (candidate === 'pt') candidate = 'pt-BR'
  return languages.some((language) => language.id === candidate)
    ? candidate
    : undefined
}

export function preferredLocale(tags: readonly string[]): string {
  for (const tag of tags) {
    const language = matchLocale(tag)
    if (language) return language
  }
  return 'en'
}

export function resolveLocale(
  preference: string,
  systemLanguage: string,
): string {
  return preferredLocale([
    preference === 'system' || !preference ? systemLanguage : preference,
  ])
}

export function localeDirection(language: string): 'ltr' | 'rtl' {
  return languages.find((entry) => entry.id === language)?.direction === 'rtl'
    ? 'rtl'
    : 'ltr'
}

// Simple interpolation only: dictionaries use count-neutral wording where needed.
// Isolate substituted values in RTL labels so paths and identifiers stay readable.
export function translate(
  language: string,
  text: string,
  values: Record<string, string | number> = {},
): string {
  const resolved = preferredLocale([language])
  const translated = dictionaries[resolved]?.[text] || text
  return translated.replace(/\{(\w+)\}/g, (placeholder, key: string) => {
    const value = values[key]
    if (value === undefined) return placeholder
    const formatted =
      typeof value === 'number'
        ? new Intl.NumberFormat(resolved).format(value)
        : value
    return localeDirection(resolved) === 'rtl'
      ? `\u2068${formatted}\u2069`
      : formatted
  })
}
