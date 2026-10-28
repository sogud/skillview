import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useState,
  type ReactNode,
} from 'react'
import {
  preferredLocale,
  resolveLocale,
  localeDirection,
  translate,
} from './locale-core'
import * as api from './api'

type LocaleContextValue = {
  language: string
  direction: 'ltr' | 'rtl'
  setLanguage: (language: string) => void
  t: (text: string, values?: Record<string, string | number>) => string
  error: (failure: unknown) => string
}

const LocaleContext = createContext<LocaleContextValue | null>(null)

export function LocaleProvider({ children }: { children: ReactNode }) {
  const [language, setLanguage] = useState('system')
  const [systemLanguage, setSystemLanguage] = useState(
    preferredLocale(
      navigator.languages.length ? navigator.languages : [navigator.language],
    ),
  )
  useEffect(() => {
    void api
      .SystemLanguage()
      .then(setSystemLanguage)
      .catch(() => {})
  }, [])
  const resolved = resolveLocale(language, systemLanguage)
  useEffect(() => {
    document.documentElement.lang = resolved
    document.documentElement.dir = localeDirection(resolved)
  }, [resolved])
  const t = useCallback(
    (text: string, values: Record<string, string | number> = {}) =>
      translate(resolved, text, values),
    [resolved],
  )
  const error = useCallback(
    (failure: unknown) => {
      const text = (
        failure instanceof Error ? failure.message : String(failure)
      ).replace(/^Error:\s*/, '')
      const retrySeconds = text.match(
        /^Too many reports\. Retry after (\d+) seconds\.$/,
      )?.[1]
      if (retrySeconds)
        return t('Too many reports. Retry after {seconds} seconds.', {
          seconds: Number(retrySeconds),
        })
      const feedbackStatus = text.match(
        /^Feedback was not accepted \(HTTP (\d+)\)\.$/,
      )?.[1]
      if (feedbackStatus)
        return t('Feedback was not accepted (HTTP {status}).', {
          status: feedbackStatus,
        })
      const youdaoCode = text.match(
        /^Youdao's free service rejected the translation \(code (\d+)\). Try again later or choose another provider\.$/,
      )?.[1]
      if (youdaoCode)
        return t(
          "Youdao's free service rejected the translation (code {code}). Try again later or choose another provider.",
          { code: youdaoCode },
        )
      const status = text.match(
        /^translation service returned HTTP (\d+)$/,
      )?.[1]
      const hints: Record<string, string> = {
        '400':
          'The translation service rejected the request. Check the selected model and provider settings.',
        '401':
          'The translation service rejected the API key. Check Translation settings.',
        '403':
          'The translation service denied access. Check your network or choose Youdao or AI translation.',
        '429':
          'The translation service is rate limited. Wait a minute, or choose another service in Translation settings.',
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
    <LocaleContext.Provider
      value={{
        language,
        direction: localeDirection(resolved),
        setLanguage,
        t,
        error,
      }}
    >
      {children}
    </LocaleContext.Provider>
  )
}

export function useLocale() {
  const locale = useContext(LocaleContext)
  if (!locale) throw new Error('LocaleProvider is missing')
  return locale
}
