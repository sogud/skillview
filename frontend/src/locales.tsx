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
      if (
        /^translation service returned HTTP \d+$/.test(text) &&
        resolved === 'zh-CN'
      ) {
        return `翻译服务返回 HTTP ${text.match(/\d+$/)?.[0]}，请检查服务与密钥。`
      }
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
