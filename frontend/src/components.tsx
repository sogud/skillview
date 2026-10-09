import { useLocale } from './locales'
import { useEffect, useRef, type ReactNode } from 'react'
import ReactMarkdown from 'react-markdown'
import remarkGfm from 'remark-gfm'
import { X, ImageIcon, type LucideIcon } from 'lucide-react'

export function IconButton({
  icon: Icon,
  label,
  active = false,
  onClick,
  disabled = false,
}: {
  icon: LucideIcon
  label: string
  active?: boolean
  onClick: () => void
  disabled?: boolean
}) {
  return (
    <button
      type="button"
      className={`icon-button${active ? ' active' : ''}`}
      title={label}
      aria-label={label}
      aria-pressed={active || undefined}
      onClick={onClick}
      disabled={disabled}
    >
      <Icon size={16} />
    </button>
  )
}

export function Modal({
  title,
  children,
  onClose,
}: {
  title: string
  children: ReactNode
  onClose: () => void
}) {
  const { t } = useLocale()
  const ref = useRef<HTMLDialogElement>(null)
  useEffect(() => {
    const dialog = ref.current!
    const previousFocus = document.activeElement as HTMLElement | null
    dialog.showModal()
    dialog.querySelector<HTMLElement>('[data-autofocus]')?.focus()
    return () => {
      dialog.close()
      previousFocus?.focus()
    }
  }, [])
  return (
    <dialog
      ref={ref}
      aria-label={title}
      onCancel={(event) => {
        event.preventDefault()
        onClose()
      }}
    >
      <div className="modal-head">
        <h2>{title}</h2>
        <IconButton icon={X} label={t("Close dialog")} onClick={onClose} />
      </div>
      {children}
    </dialog>
  )
}

export function Markdown({ text, onLink }: { text: string; onLink: (url: string) => void }) {
  const { t } = useLocale()
  return (
    <div className="markdown">
      <ReactMarkdown
        skipHtml
        remarkPlugins={[remarkGfm]}
        components={{
          a: ({ href, children }) => (
            <a
              href={href || '#'}
              onClick={(event) => {
                event.preventDefault()
                if (href) onLink(href)
              }}
            >
              {children}
            </a>
          ),
          img: ({ src, alt }) => (
            <button
              className="image-reference"
              onClick={() => {
                if (typeof src === 'string') onLink(src)
              }}
            >
              <ImageIcon size={14} />
              {alt || t('View image resource')}
            </button>
          ),
        }}
      >
        {text}
      </ReactMarkdown>
    </div>
  )
}
