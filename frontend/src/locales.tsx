import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useState,
  type ReactNode,
} from 'react'
import messages from '../../internal/locale/messages.json'
import * as api from './api'

type LocaleContextValue = {
  language: string
  setLanguage: (language: string) => void
  t: (text: string, values?: Record<string, string | number>) => string
  error: (failure: unknown) => string
}

const LocaleContext = createContext<LocaleContextValue | null>(null)

export function LocaleProvider({ children }: { children: ReactNode }) {
  const [language, setLanguage] = useState('system')
  const [systemLanguage, setSystemLanguage] = useState(
    navigator.language.startsWith('zh') ? 'zh-CN' : 'en',
  )
  useEffect(() => {
    void api
      .SystemLanguage()
      .then(setSystemLanguage)
      .catch(() => {})
  }, [])
  const resolved = language === 'system' ? systemLanguage : language
  useEffect(() => {
    document.documentElement.lang = resolved
  }, [resolved])
  const t = useCallback(
    (text: string, values: Record<string, string | number> = {}) => {
      let translated =
        resolved === 'zh-CN'
          ? ((messages['zh-CN'] as Record<string, string>)[text] ?? text)
          : text
      for (const [key, value] of Object.entries(values)) {
        translated = translated.replaceAll(`{${key}}`, String(value))
      }
      return translated
    },
    [resolved],
  )
  const error = useCallback(
    (failure: unknown) => {
      const text = (
        failure instanceof Error ? failure.message : String(failure)
      ).replace(/^Error:\s*/, '')
      const status = text.match(
        /^translation service returned HTTP (\d+)$/,
      )?.[1]
      const hints: Record<string, string> = {
        '400':
          'The translation service rejected the request. Check the selected model and provider settings.',
        '401':
          'The translation service rejected the API key. Check Translation settings.',
        '403':
          'The translation service denied access. Check your network or choose Google Cloud or AI translation.',
        '429':
          'The translation service is rate limited. Wait and retry, or choose another provider.',
      }
      if (status && hints[status]) return `HTTP ${status} · ${t(hints[status])}`
      if (status)
        return t(
          'Translation service returned HTTP {status}. Try again or choose another provider.',
          { status },
        )
      return t(text)
    },
    [resolved, t],
  )
  return (
    <LocaleContext.Provider value={{ language, setLanguage, t, error }}>
      {children}
    </LocaleContext.Provider>
  )
}

export function useLocale() {
  const locale = useContext(LocaleContext)
  if (!locale) throw new Error('LocaleProvider is missing')
  return locale
}
