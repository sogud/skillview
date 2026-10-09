import {
  useEffect,
  useId,
  useLayoutEffect,
  useRef,
  useState,
  type CSSProperties,
} from 'react'
import { createPortal } from 'react-dom'
import { Check, ChevronDown } from 'lucide-react'

type Option = { value: string; label: string; description?: string }

export function Select({
  label,
  value,
  options,
  onChange,
}: {
  label: string
  value: string
  options: Option[]
  onChange: (value: string) => void
}) {
  const id = useId()
  const trigger = useRef<HTMLButtonElement>(null)
  const menu = useRef<HTMLDivElement>(null)
  const [open, setOpen] = useState(false)
  const [active, setActive] = useState(0)
  const [position, setPosition] = useState<CSSProperties>({})
  const selected = options.findIndex((option) => option.value === value)

  function show() {
    setActive(Math.max(selected, 0))
    setOpen(true)
  }
  function close(focus = true) {
    setOpen(false)
    if (focus) trigger.current?.focus()
  }
  function choose(index: number) {
    if (options[index]) onChange(options[index].value)
    close()
  }
  useLayoutEffect(() => {
    if (!open || !trigger.current) return
    const rect = trigger.current.getBoundingClientRect()
    const below = window.innerHeight - rect.bottom - 12
    const above = rect.top - 12
    const upwards = below < 160 && above > below
    const width = Math.min(Math.max(rect.width, 260), window.innerWidth - 24)
    setPosition({
      width,
      left: Math.min(rect.left, window.innerWidth - width - 12),
      top: upwards ? undefined : rect.bottom + 5,
      bottom: upwards ? window.innerHeight - rect.top + 5 : undefined,
      maxHeight: Math.min(280, upwards ? above : below),
    })
    menu.current?.focus()
  }, [open])
  useEffect(() => {
    if (!open) return
    function outside(event: PointerEvent) {
      if (
        !trigger.current?.contains(event.target as Node) &&
        !menu.current?.contains(event.target as Node)
      )
        setOpen(false)
    }
    function dismiss() {
      setOpen(false)
    }
    function scroll(event: Event) {
      if (!menu.current?.contains(event.target as Node)) dismiss()
    }
    document.addEventListener('pointerdown', outside, true)
    document.addEventListener('scroll', scroll, true)
    window.addEventListener('resize', dismiss)
    return () => {
      document.removeEventListener('pointerdown', outside, true)
      document.removeEventListener('scroll', scroll, true)
      window.removeEventListener('resize', dismiss)
    }
  }, [open])
  useEffect(() => {
    if (open)
      menu.current?.children[active]?.scrollIntoView({ block: 'nearest' })
  }, [active, open])
  return (
    <>
      <button
        ref={trigger}
        type="button"
        role="combobox"
        className="select-trigger"
        aria-label={label}
        aria-expanded={open}
        aria-controls={`${id}-menu`}
        aria-haspopup="listbox"
        onClick={() => (open ? close() : show())}
        onKeyDown={(event) => {
          if (['ArrowDown', 'ArrowUp', 'Home', 'End'].includes(event.key)) {
            event.preventDefault()
            show()
          }
        }}
      >
        <span>{options[selected]?.label ?? label}</span>
        <ChevronDown size={14} />
      </button>
      {open &&
        createPortal(
          <div
            ref={menu}
            id={`${id}-menu`}
            className="select-menu"
            role="listbox"
            tabIndex={-1}
            aria-label={label}
            aria-activedescendant={`${id}-${active}`}
            style={position}
            onKeyDown={(event) => {
              const key = event.key
              if (key === 'Tab') {
                close(false)
                return
              }
              if (key === 'Escape') {
                event.preventDefault()
                event.stopPropagation()
                close()
                return
              }
              if (
                ['ArrowDown', 'ArrowUp', 'Home', 'End', 'Enter', ' '].includes(
                  key,
                )
              ) {
                event.preventDefault()
                if (key === 'Enter' || key === ' ') choose(active)
                else if (key === 'Home') setActive(0)
                else if (key === 'End') setActive(options.length - 1)
                else
                  setActive((index) =>
                    Math.max(
                      0,
                      Math.min(
                        options.length - 1,
                        index + (key === 'ArrowDown' ? 1 : -1),
                      ),
                    ),
                  )
              } else if (key.length === 1) {
                const index = options.findIndex(
                  (option, index) =>
                    index > active &&
                    option.label
                      .toLocaleLowerCase()
                      .startsWith(key.toLocaleLowerCase()),
                )
                const first = options.findIndex((option) =>
                  option.label
                    .toLocaleLowerCase()
                    .startsWith(key.toLocaleLowerCase()),
                )
                if (index >= 0 || first >= 0)
                  setActive(index >= 0 ? index : first)
              }
            }}
          >
            {options.map((option, index) => (
              <div
                key={option.value}
                id={`${id}-${index}`}
                role="option"
                aria-selected={option.value === value}
                className={`select-option${active === index ? ' focused' : ''}`}
                title={option.description}
                onPointerMove={() => setActive(index)}
                onClick={() => choose(index)}
              >
                <span>
                  <span>{option.label}</span>
                  {option.description && <small>{option.description}</small>}
                </span>
                {option.value === value && <Check size={14} />}
              </div>
            ))}
          </div>,
          trigger.current?.closest('dialog') ?? document.body,
        )}
    </>
  )
}
