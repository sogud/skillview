import { useLocale } from './locales'
import {
  FileText,
  Pencil,
  Folder,
  X,
  Save,
  Eye,
  ArrowLeft,
  FileCode,
  Link,
  Lock,
  AlertCircle,
  Languages,
  Trash2,
  Archive,
  Copy,
  Download,
  PanelLeftClose,
  PanelLeftOpen,
  Minus,
  Plus,
  Check,
  LoaderCircle,
} from 'lucide-react'
import {
  lazy,
  Suspense,
  useEffect,
  useRef,
  useState,
  type CSSProperties,
} from 'react'
import { Events } from '@wailsio/runtime'
import { IconButton, Markdown, Disclosure } from './components'
import { Select } from './Select'
import { FileTree } from './FileTree'
import * as api from './api'
import type {
  Document,
  Skill,
  Resource,
  TranslationResult,
  Occurrence,
  TranslationProgress,
} from './api'

const Editor = lazy(() => import('./Editor'))

function translationProviderName(provider: string): string {
  const names: Record<string, string> = {
    google: 'Google',
    'google-cloud': 'Google Cloud',
    ai: 'AI',
    youdao: 'Youdao',
    libretranslate: 'LibreTranslate',
  }
  return names[provider] ?? provider
}

export type Draft = { base: Document; text: string }

type ReaderProps = {
  fontSize: number
  explorerWidth: number
  onLayoutChange: (fontSize: number, width: number) => void
  translationVersion: number
  translationDisplayVersion: number
  onConfigureTranslation: () => void
  onReportTranslation: () => void
  onNewResource: () => void
  onImportResource: () => void
  onRenameResource: (resource: Resource) => void
  onTrashResource: (resource: Resource) => void
  onTrashSkill: () => void
  onArchiveSkill: () => void
  onRemoveLink: (copy: Skill, occurrence: Occurrence) => void
  skill: Skill
  copies: Skill[]
  versions: number
  onCopy: (copy: Skill) => void
  document: Document | null
  resources: Resource[]
  draft: Draft | null
  raw: boolean
  loading: boolean
  error: string
  saving: boolean
  onRaw: () => void
  onEdit: () => void
  onChange: (text: string) => void
  onSave: () => void
  onCancel: () => void
  onResource: (path: string) => void
  onLink: (url: string) => void
  onReveal: () => void
  onClose: () => void
}

export function Reader(props: ReaderProps) {
  const { t, error: translationErrorMessage, direction } = useLocale()
  const { skill, document: doc, draft } = props
  const [translation, setTranslation] = useState<TranslationResult | null>(null)
  const [translated, setTranslated] = useState(false)
  const [displayMode, setDisplayMode] = useState('immersive')
  useEffect(() => {
    let active = true
    void api
      .GetTranslationSettings()
      .then((settings) => {
        if (active) setDisplayMode(settings.displayMode || 'immersive')
      })
      .catch(() => {})
    return () => {
      active = false
    }
  }, [props.translationDisplayVersion])
  const [translating, setTranslating] = useState(false)
  const [progress, setProgress] = useState<TranslationProgress | null>(null)
  const [elapsed, setElapsed] = useState(0)
  const [translationError, setTranslationError] = useState('')
  const [translationNotice, setTranslationNotice] = useState('')
  const [exporting, setExporting] = useState(false)
  const [explorerOpen, setExplorerOpen] = useState(true)
  const [width, setWidth] = useState(props.explorerWidth)
  const [maxWidth, setMaxWidth] = useState(400)
  const restoreTree = useRef<HTMLButtonElement>(null)
  const focusExplorer = useRef(false)
  const body = useRef<HTMLDivElement>(null)
  const content = useRef<HTMLDivElement>(null)
  const drag = useRef<{ x: number; width: number } | null>(null)
  const pending = useRef<ReturnType<typeof api.Translate> | null>(null)
  const sequence = useRef(0)
  const requestID = useRef('')
  function showExplorer(open: boolean) {
    focusExplorer.current = true
    setExplorerOpen(open)
  }
  useEffect(() => {
    if (!focusExplorer.current) return
    focusExplorer.current = false
    if (explorerOpen) {
      body.current
        ?.querySelector<HTMLButtonElement>(
          '[role="treeitem"][aria-selected="true"]',
        )
        ?.focus()
    } else {
      restoreTree.current?.focus()
    }
  }, [explorerOpen])
  useEffect(() => setWidth(props.explorerWidth), [props.explorerWidth])
  useEffect(() => {
    content.current?.scrollTo(0, 0)
  }, [doc?.skillId, doc?.path])
  useEffect(() => {
    const observer = new ResizeObserver(([entry]) => {
      setMaxWidth(
        Math.max(
          180,
          Math.min(
            400,
            Math.floor(entry.contentRect.width * 0.38),
            Math.floor(entry.contentRect.width) - 320,
          ),
        ),
      )
    })
    if (body.current) observer.observe(body.current)
    return () => observer.disconnect()
  }, [])
  function resize(value: number) {
    return Math.max(180, Math.min(maxWidth, Math.round(value)))
  }
  function saveWidth(value: number) {
    setWidth(value)
    props.onLayoutChange(props.fontSize, value)
  }
  useEffect(
    () =>
      Events.On('skillview:translation', (event) => {
        const next = event.data as TranslationProgress
        if (requestID.current && next.requestId === requestID.current)
          setProgress(next)
      }),
    [],
  )
  useEffect(() => {
    if (!translating) return
    const start = Date.now()
    const timer = window.setInterval(
      () => setElapsed(Math.floor((Date.now() - start) / 1000)),
      1000,
    )
    return () => window.clearInterval(timer)
  }, [translating])
  useEffect(() => {
    sequence.current++
    pending.current?.cancel()
    pending.current = null
    requestID.current = ''
    setProgress(null)
    setTranslation(null)
    setTranslated(false)
    setTranslating(false)
    setTranslationError('')
    setTranslationNotice('')
    return () => {
      sequence.current++
      pending.current?.cancel()
    }
  }, [doc?.skillId, doc?.path, doc?.revision, props.translationVersion])
  useEffect(() => {
    if (!translationNotice) return
    const timeout = window.setTimeout(() => setTranslationNotice(''), 2800)
    return () => window.clearTimeout(timeout)
  }, [translationNotice])
  const canTranslate =
    !!doc && (doc.kind === 'markdown' || doc.kind === 'text') && !draft
  async function translate() {
    if (!doc || translating) return
    if (translation) {
      setTranslated((value) => !value)
      return
    }
    const current = ++sequence.current
    setTranslating(true)
    setElapsed(0)
    setProgress(null)
    setTranslationNotice('')
    setTranslationError('')
    try {
      const settings = await api.GetTranslationSettings()
      if (current !== sequence.current) return
      setDisplayMode(settings.displayMode || 'immersive')
      if (
        (settings.provider === 'ai' && !settings.model.trim()) ||
        (settings.provider === 'libretranslate' && !settings.libreUrl.trim())
      ) {
        props.onConfigureTranslation()
        return
      }
      requestID.current = `${doc.skillId}:${current}:${Date.now()}`
      const request = api.Translate(
        doc.skillId,
        doc.path,
        doc.revision,
        requestID.current,
      )
      pending.current = request
      const result = await request
      if (current === sequence.current) {
        setTranslation(result)
        setTranslated(true)
      }
    } catch (failure) {
      if (current === sequence.current)
        setTranslationError(translationErrorMessage(failure))
    } finally {
      if (current === sequence.current) {
        setTranslating(false)
        pending.current = null
        requestID.current = ''
      }
    }
  }
  function cancelTranslation() {
    sequence.current++
    pending.current?.cancel()
    pending.current = null
    requestID.current = ''
    setTranslating(false)
    setTranslationError('')
    setTranslationNotice(t('Translation canceled'))
  }
  async function exportTranslation() {
    if (!doc || !translation || exporting) return
    setExporting(true)
    setTranslationError('')
    try {
      if (await api.ExportTranslation(doc.skillId, doc.path, translation))
        setTranslationNotice(t('Translation exported'))
    } catch (failure) {
      setTranslationError(translationErrorMessage(failure))
    } finally {
      setExporting(false)
    }
  }
  async function copyTranslation() {
    if (!doc || !translation) return
    setTranslationError('')
    try {
      await api.CopyTranslation(doc.skillId, doc.path, translation)
      setTranslationNotice(t('Translation copied'))
    } catch (failure) {
      setTranslationError(translationErrorMessage(failure))
    }
  }
  const dirty = !!draft && draft.text !== draft.base.text
  return (
    <aside
      className="reader"
      aria-label={`${skill.name} ${t('reader')}`}
      style={{ '--reader-font-size': `${props.fontSize}px` } as CSSProperties}
    >
      <nav className="reader-navigation" aria-label={t('Skill navigation')}>
        <button
          className="reader-back"
          onClick={props.onClose}
          disabled={props.saving}
        >
          <ArrowLeft size={15} />
          {t('Back to Skills')}
        </button>
      </nav>
      <div className="reader-body" ref={body}>
        {explorerOpen && (
          <nav
            id="skill-file-explorer"
            className="file-explorer"
            aria-label={t('Skill files')}
            style={{ width: Math.min(width, maxWidth) }}
          >
            <div className="explorer-head">
              <span className="explorer-title" title={skill.path}>
                {skill.name}
              </span>
              <IconButton
                icon={PanelLeftClose}
                label={t('Hide file tree')}
                onClick={() => showExplorer(false)}
              />
            </div>
            {props.copies.length > 1 && (
              <div className="explorer-copy">
                <Select
                  label={t('Skill copy')}
                  value={skill.id}
                  options={props.copies.map((copy) => ({
                    value: copy.id,
                    label:
                      (copy.tools ?? [])
                        .map((tool) =>
                          t(
                            tool === 'Shared'
                              ? 'Common Skills'
                              : tool === 'Local'
                                ? 'Local files'
                                : tool,
                          ),
                        )
                        .join(' · ') || t('Local files'),
                    description: copy.path,
                    descriptionDirection: 'ltr' as const,
                  }))}
                  onChange={(id) => {
                    const copy = props.copies.find((copy) => copy.id === id)
                    if (copy) props.onCopy(copy)
                  }}
                />
                <small>
                  {t('{count} copies', { count: props.copies.length })}
                  {props.versions > 1
                    ? ` · ${t('{count} versions', { count: props.versions })}`
                    : ''}
                </small>
              </div>
            )}
            {skill.occurrences?.some((entry) => entry.removable) && (
              <Disclosure
                label={t('Linked entries')}
                className="explorer-links"
              >
                {skill.occurrences
                  .filter((entry) => entry.removable)
                  .map((entry) => (
                    <div
                      key={entry.sourceId + entry.path}
                      className="link-entry"
                    >
                      <Link size={12} />
                      <span dir="ltr" title={entry.path}>{entry.path}</span>
                      <IconButton
                        icon={Trash2}
                        label={t('Remove link ({path})', { path: entry.path })}
                        onClick={() => props.onRemoveLink(skill, entry)}
                      />
                    </div>
                  ))}
              </Disclosure>
            )}
            <FileTree
              key={skill.id}
              resources={props.resources}
              path={doc?.path ?? 'SKILL.md'}
              editable={skill.editable}
              onOpen={props.onResource}
              onRename={props.onRenameResource}
              onTrash={props.onTrashResource}
              onNew={props.onNewResource}
              onImport={props.onImportResource}
            />
          </nav>
        )}
        {explorerOpen && (
          <div
            className="explorer-resize"
            role="separator"
            tabIndex={0}
            aria-label={t('Resize file tree')}
            aria-orientation="vertical"
            aria-controls="skill-file-explorer"
            aria-valuemin={180}
            aria-valuemax={maxWidth}
            aria-valuenow={Math.min(width, maxWidth)}
            title={t(
              'Drag to resize. Double-click to reset. Use arrow keys to adjust.',
            )}
            onPointerDown={(event) => {
              if (event.button !== 0) return
              event.preventDefault()
              event.currentTarget.focus()
              event.currentTarget.setPointerCapture(event.pointerId)
              drag.current = {
                x: event.clientX,
                width: Math.min(width, maxWidth),
              }
            }}
            onPointerMove={(event) => {
              if (drag.current)
                setWidth(
                  resize(drag.current.width + (event.clientX - drag.current.x) * (direction === 'rtl' ? -1 : 1)),
                )
            }}
            onPointerUp={() => {
              if (drag.current) saveWidth(resize(width))
              drag.current = null
            }}
            onPointerCancel={() => {
              drag.current = null
              setWidth(props.explorerWidth)
            }}
            onLostPointerCapture={() => {
              drag.current = null
            }}
            onDoubleClick={() => saveWidth(244)}
            onKeyDown={(event) => {
              if (
                !['ArrowLeft', 'ArrowRight', 'Home', 'End', 'Enter'].includes(
                  event.key,
                )
              )
                return
              event.preventDefault()
              let next = Math.min(width, maxWidth)
              if (event.key === 'Home') next = 180
              else if (event.key === 'End') next = maxWidth
              else if (event.key === 'Enter') next = 244
              else
                next +=
                  (event.key === 'ArrowRight' ? 1 : -1) *
                  (direction === 'rtl' ? -1 : 1) *
                  (event.shiftKey ? 40 : 16)
              saveWidth(resize(next))
            }}
          />
        )}
        <section
          className="reader-document"
          aria-label={doc?.path ?? 'SKILL.md'}
        >
          <header className="reader-head">
            {!explorerOpen && (
              <button
                className="reader-tree-restore"
                ref={restoreTree}
                onClick={() => showExplorer(true)}
              >
                <PanelLeftOpen size={14} />
                {t('Show file tree')}
              </button>
            )}

            <span className="filename" title={doc?.path}>
              {doc?.path ?? 'SKILL.md'}
              {dirty && (
                <span className="unsaved" aria-label={t('Unsaved changes')}>
                  ●
                </span>
              )}
            </span>
            <div className="reader-tools">
              {canTranslate && (
                <button
                  className="reader-translate"
                  aria-pressed={translated}
                  title={t('Translation does not change the original file.')}
                  onClick={() => {
                    if (translating) cancelTranslation()
                    else void translate()
                  }}
                >
                  {translating ? <X size={14} /> : <Languages size={15} />}
                  {t(
                    translating
                      ? 'Cancel translation'
                      : translated
                        ? 'Show original'
                        : translation
                          ? doc?.kind === 'markdown' &&
                            displayMode === 'immersive'
                            ? 'Show bilingual translation'
                            : 'Show translation'
                          : 'Translate document',
                  )}
                </button>
              )}
              {doc &&
                !draft &&
                (doc.kind === 'markdown' || doc.kind === 'text') && (
                  <IconButton
                    icon={props.raw ? Eye : FileCode}
                    label={props.raw ? t('Read Markdown') : t('View raw file')}
                    active={props.raw}
                    onClick={() => {
                      setTranslated(false)
                      props.onRaw()
                    }}
                  />
                )}
              {doc?.editable && !draft && (
                <IconButton
                  icon={Pencil}
                  label={t('Edit file')}
                  onClick={() => {
                    setTranslated(false)
                    props.onEdit()
                  }}
                />
              )}
              {!draft && (
                <IconButton
                  icon={Archive}
                  label={t(
                    skill.editable
                      ? 'Archive Skill'
                      : 'Archive Skill (allow editing for the original folder first)',
                  )}
                  disabled={!skill.editable}
                  onClick={props.onArchiveSkill}
                />
              )}
              {skill.editable && !draft && (
                <IconButton
                  icon={Trash2}
                  label={t('Move Skill to Trash')}
                  onClick={props.onTrashSkill}
                />
              )}
              <IconButton
                icon={Folder}
                label={t('Show in Finder')}
                onClick={props.onReveal}
              />
            </div>
          </header>
          {translating && (
            <div
              className="translation-progress"
              role="status"
              aria-live="polite"
            >
              <span>
                <LoaderCircle
                  size={13}
                  className="translation-spinner"
                  aria-hidden="true"
                />
                {progress
                  ? t('Translating {done} / {total} sections', {
                      done: progress.completed,
                      total: progress.total,
                    })
                  : t('Preparing translation…')}
                {' · '}
                {t('{seconds}s elapsed', { seconds: elapsed })}
                {progress
                  ? ` · ${translationProviderName(progress.provider)} → ${progress.targetLanguage}`
                  : ''}
              </span>
              <progress
                aria-label={t('Translation progress')}
                value={
                  progress && progress.total > 0
                    ? progress.completed
                    : undefined
                }
                max={progress?.total || 1}
              />
              {progress && progress.completed === progress.total && (
                <span>{t('Checking the complete translation…')}</span>
              )}
            </div>
          )}
          {translationError && (
            <div className="translation-error" role="alert">
              <AlertCircle size={14} />
              <span>{translationError}</span>
              <button onClick={props.onReportTranslation}>
                {t('Report this issue')}
              </button>
              {!translation && (
                <button
                  onClick={() => {
                    void translate()
                  }}
                >
                  {t('Try again')}
                </button>
              )}
              {!translated && (
                <button onClick={props.onConfigureTranslation}>
                  {t('Configure translation')}
                </button>
              )}
            </div>
          )}
          {translation && !draft && (
            <p
              className="translation-note"
              title={t('Translation does not change the original file.')}
            >
              <strong>
                <Check size={13} aria-hidden="true" />
                {translation.sections === 0
                  ? t('No prose to translate')
                  : t(
                      translated ? 'Translation complete' : 'Translation ready',
                    )}
              </strong>
              {' · '}
              {translationProviderName(translation.provider)} →{' '}
              {translation.targetLanguage}
              {translation.cached ? ` · ${t('Cached translation')}` : ''}
              <span className="translation-actions">
                <IconButton
                  icon={Copy}
                  label={t('Copy translation ({language})', {
                    language: translation.targetLanguage,
                  })}
                  onClick={() => {
                    void copyTranslation()
                  }}
                  disabled={exporting}
                />
                <IconButton
                  icon={Download}
                  label={t('Export translation')}
                  onClick={() => {
                    void exportTranslation()
                  }}
                  disabled={exporting}
                />
              </span>
            </p>
          )}
          {translationNotice && (
            <p className="quiet-note translation-status" role="status">
              {translationNotice}
            </p>
          )}
          <div
            ref={content}
            className={`reader-content${draft ? ' editing' : ''}`}
            tabIndex={draft ? -1 : 0}
          >
            {props.loading ? (
              <p className="quiet-note">{t('Reading file\u2026')}</p>
            ) : props.error ? (
              <div className="file-error">
                <AlertCircle size={20} />
                <p>{props.error}</p>
                <button
                  onClick={() => props.onResource(doc?.path ?? 'SKILL.md')}
                >
                  {t('Try again')}
                </button>
              </div>
            ) : draft ? (
              <>
                <Suspense
                  fallback={
                    <p className="quiet-note">{t('Opening editor\u2026')}</p>
                  }
                >
                  <Editor
                    key={`${draft.base.skillId}:${draft.base.path}:${draft.base.revision}`}
                    text={draft.text}
                    newline={draft.base.newline}
                    onChange={props.onChange}
                  />
                </Suspense>
              </>
            ) : doc ? (
              <>
                {doc.path === 'SKILL.md' && (
                  <div className="skill-intro" dir="auto">
                    {!translated && !props.raw && !/^\s*#\s/.test(doc.body) && (
                      <>
                        <h1>{skill.name}</h1>
                        {skill.description && <p>{skill.description}</p>}
                      </>
                    )}
                    {(skill.warnings ?? []).length > 0 && (
                      <Disclosure
                        className="parse-warning"
                        label={
                          <>
                            <AlertCircle size={12} />{' '}
                            {t('Metadata needs attention')}
                          </>
                        }
                      >
                        {skill.warnings?.map((warning) => (
                          <p key={warning}>{warning}</p>
                        ))}
                      </Disclosure>
                    )}
                  </div>
                )}
                {doc.path !== 'SKILL.md' && !explorerOpen && (
                  <button
                    className="back-to-skill"
                    onClick={() => props.onResource('SKILL.md')}
                  >
                    <ArrowLeft size={12} />
                    SKILL.md
                  </button>
                )}
                {translated && translation ? (
                  doc.kind === 'markdown' ? (
                    <Markdown
                      text={
                        displayMode === 'translation'
                          ? translation.text
                          : doc.body
                      }
                      translation={
                        displayMode === 'translation' ? undefined : translation
                      }
                      onLink={props.onLink}
                    />
                  ) : (
                    <div className="plain-translation">
                      {displayMode !== 'translation' && (
                        <section aria-label={t('Original')}>
                          <h3>{t('Original')}</h3>
                          <pre className="raw-file" dir="auto">
                            {doc.body}
                          </pre>
                        </section>
                      )}
                      <section
                        aria-label={t('Translation')}
                        lang={translation.targetLanguage}
                      >
                        {displayMode !== 'translation' && (
                          <h3>{t('Translation')}</h3>
                        )}
                        <pre className="raw-file" dir="auto">
                          {translation.text}
                        </pre>
                      </section>
                    </div>
                  )
                ) : doc.kind === 'image' ? (
                  <img
                    className="resource-image"
                    src={doc.dataUrl}
                    alt={doc.path}
                  />
                ) : doc.kind === 'binary' ? (
                  <div className="binary-info">
                    <FileText size={28} />
                    <p>{doc.path}</p>
                    <span>{formatSize(doc.size)}</span>
                    <p>
                      {translationErrorMessage(
                        doc.readOnlyReason ||
                          'Preview is unavailable for this file type.',
                      )}
                    </p>
                    <button onClick={props.onReveal}>
                      {t('Show in Finder')}
                    </button>
                  </div>
                ) : props.raw || doc.kind !== 'markdown' ? (
                  <pre className="raw-file">{doc.text}</pre>
                ) : (
                  <Markdown text={doc.body} onLink={props.onLink} />
                )}
              </>
            ) : null}
          </div>
          {draft && (
            <div className="edit-actions">
              <span>
                {props.saving
                  ? t('Saving…')
                  : dirty
                    ? t('Unsaved changes')
                    : t('No changes')}
              </span>
              <button onClick={props.onCancel} disabled={props.saving}>
                {t('Done')}
              </button>
              <button
                className="primary"
                onClick={props.onSave}
                disabled={!dirty || props.saving}
              >
                <Save size={13} />
                {t('Save')}
                <kbd>⌘S</kbd>
              </button>
            </div>
          )}
          <footer className="reader-footer">
            <Lock size={11} />
            <span className="access-label">
              {doc?.editable ? t('Writable') : t('Read-only')}
            </span>
            {!draft && (
              <div
                className="reading-size"
                role="group"
                aria-label={t('Reading size')}
              >
                <IconButton
                  icon={Minus}
                  label={t('Decrease reading size')}
                  disabled={props.fontSize <= 12}
                  onClick={() =>
                    props.onLayoutChange(
                      props.fontSize - 1,
                      props.explorerWidth,
                    )
                  }
                />
                <button
                  className="reading-size-value"
                  title={t('Reset reading size')}
                  aria-label={t('Reading size {size}px. Reset to 14px.', {
                    size: props.fontSize,
                  })}
                  onClick={() => props.onLayoutChange(14, props.explorerWidth)}
                >
                  {props.fontSize}
                </button>
                <IconButton
                  icon={Plus}
                  label={t('Increase reading size')}
                  disabled={props.fontSize >= 20}
                  onClick={() =>
                    props.onLayoutChange(
                      props.fontSize + 1,
                      props.explorerWidth,
                    )
                  }
                />
              </div>
            )}
          </footer>
        </section>
      </div>
    </aside>
  )
}

function formatSize(size: number) {
  return size < 1024
    ? `${size} B`
    : size < 1024 * 1024
      ? `${(size / 1024).toFixed(1)} KB`
      : `${(size / 1024 / 1024).toFixed(1)} MB`
}
