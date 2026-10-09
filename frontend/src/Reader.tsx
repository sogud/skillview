import { useLocale } from './locales'
import {
  FileText,
  Pencil,
  Folder,
  X,
  Save,
  Eye,
  ArrowLeft,
  Image,
  FileCode,
  Link,
  Lock,
  AlertCircle,
  Languages,
  Plus,
  Upload,
  Trash2,
  FilePenLine,
  Copy,
  Download,
} from 'lucide-react'
import { lazy, Suspense, useEffect, useRef, useState } from 'react'
import { Events } from '@wailsio/runtime'
import { IconButton, Markdown } from './components'
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

export type Draft = { base: Document; text: string }

type ReaderProps = {
  translationVersion: number
  onConfigureTranslation: () => void
  onNewResource: () => void
  onImportResource: () => void
  onRenameResource: (resource: Resource) => void
  onTrashResource: (resource: Resource) => void
  onTrashSkill: () => void
  onRemoveLink: (copy: Skill, occurrence: Occurrence) => void
  skill: Skill
  copies: Skill[]
  versions: number
  onCopy: (copy: Skill) => void
  document: Document | null
  resources: Resource[]
  draft: Draft | null
  tab: 'document' | 'resources'
  raw: boolean
  loading: boolean
  error: string
  saving: boolean
  onTab: (tab: 'document' | 'resources') => void
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
  const { t, error: translationErrorMessage } = useLocale()
  const { skill, document: doc, draft } = props
  const [translation, setTranslation] = useState<TranslationResult | null>(null)
  const [translated, setTranslated] = useState(false)
  const [translating, setTranslating] = useState(false)
  const [progress, setProgress] = useState<TranslationProgress | null>(null)
  const [elapsed, setElapsed] = useState(0)
  const [translationError, setTranslationError] = useState('')
  const [translationNotice, setTranslationNotice] = useState('')
  const [exporting, setExporting] = useState(false)
  const [copiesOpen, setCopiesOpen] = useState(false)
  const pending = useRef<ReturnType<typeof api.Translate> | null>(null)
  const sequence = useRef(0)
  const requestID = useRef('')
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
    !!doc &&
    (doc.kind === 'markdown' || doc.kind === 'text') &&
    !draft &&
    props.tab === 'document'
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
      if (
        (settings.provider === 'ai' && !settings.model.trim()) ||
        (settings.provider === 'google-cloud' && !settings.hasGoogleKey)
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
    <aside className="reader" aria-label={`${skill.name} ${t('reader')}`}>
      <header className="reader-head">
        <button
          className="reader-back"
          onClick={props.onClose}
          disabled={props.saving}
        >
          <ArrowLeft size={15} />
          {t('Back to Skills')}
        </button>
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
                      ? 'Show translation'
                      : 'Translate document',
              )}
            </button>
          )}
          {doc &&
            props.tab === 'document' &&
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
      <div
        className="reader-tabs"
        role="tablist"
        aria-label={t('Skill sections')}
      >
        <button
          role="tab"
          aria-selected={props.tab === 'document'}
          onClick={() => props.onTab('document')}
        >
          {t('Document')}
        </button>
        <button
          role="tab"
          aria-selected={props.tab === 'resources'}
          onClick={() => props.onTab('resources')}
        >
          {t('Resources')}
          <span>{props.resources.length}</span>
        </button>
      </div>
      {(props.copies.length > 1 ||
        props.copies.some((copy) =>
          copy.occurrences?.some((entry) => entry.removable),
        )) && (
        <details
          className="skill-copies"
          open={copiesOpen}
          onToggle={(event) => setCopiesOpen(event.currentTarget.open)}
        >
          <summary>
            {t('{count} copies', { count: props.copies.length })}
          </summary>
          {copiesOpen && (
            <div className="copy-list">
              <p>{t('Only the selected copy is edited.')}</p>
              {props.versions > 1 && (
                <p>
                  {t('These copies contain {count} different versions.', {
                    count: props.versions,
                  })}
                </p>
              )}
              {props.copies.map((copy) => (
                <div key={copy.id} className="copy-entry">
                  <button
                    aria-pressed={copy.id === skill.id}
                    title={copy.path}
                    onClick={() => props.onCopy(copy)}
                  >
                    <strong>
                      {(copy.tools ?? [])
                        .map((tool) =>
                          t(
                            tool === 'Shared'
                              ? 'Shared Skills'
                              : tool === 'Local'
                                ? 'Local files'
                                : tool,
                          ),
                        )
                        .join(' · ')}
                    </strong>
                    <span>{copy.path}</span>
                  </button>
                  {copy.occurrences
                    ?.filter((entry) => entry.removable)
                    .map((entry) => (
                      <div
                        key={entry.sourceId + entry.path}
                        className="link-entry"
                      >
                        <Link size={12} />
                        <span title={entry.path}>{entry.path}</span>
                        <IconButton
                          icon={Trash2}
                          label={t('Remove link ({path})', {
                            path: entry.path,
                          })}
                          onClick={() => props.onRemoveLink(copy, entry)}
                        />
                      </div>
                    ))}
                </div>
              ))}
            </div>
          )}
        </details>
      )}
      {translating && (
        <div className="translation-progress" role="status" aria-live="polite">
          <span>
            {progress
              ? t('Translating {done} / {total} sections', {
                  done: progress.completed,
                  total: progress.total,
                })
              : t('Preparing translation…')}
            {' · '}
            {t('{seconds}s elapsed', { seconds: elapsed })}
            {progress
              ? ` · ${progress.provider === 'ai' ? 'AI' : 'Google'} → ${progress.targetLanguage}`
              : ''}
          </span>
          <progress
            aria-label={t('Translation progress')}
            value={
              progress && progress.total > 0 ? progress.completed : undefined
            }
            max={progress?.total || 1}
          />
          {progress && progress.completed === progress.total && (
            <span>{t('Checking the complete translation…')}</span>
          )}
        </div>
      )}
      {translationError && (
        <p className="translation-error" role="alert">
          {translationError}{' '}
          <button
            onClick={() => {
              void translate()
            }}
          >
            {t('Try again')}
          </button>
          {!translated && (
            <button onClick={props.onConfigureTranslation}>
              {t('Configure translation')}
            </button>
          )}
        </p>
      )}
      {translated && translation && (
        <p
          className="translation-note"
          title={t('Translation does not change the original file.')}
        >
          <strong>
            {translation.sections === 0
              ? t('No prose to translate')
              : t('Translation complete')}
          </strong>
          {' · '}
          {translation.provider === 'ai' ? 'AI' : 'Google'} →{' '}
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
      <div className={`reader-content${draft ? ' editing' : ''}`}>
        {props.loading ? (
          <p className="quiet-note">{t('Reading file\u2026')}</p>
        ) : props.error ? (
          <div className="file-error">
            <AlertCircle size={20} />
            <p>{props.error}</p>
            <button onClick={() => props.onResource(doc?.path ?? 'SKILL.md')}>
              {t('Try again')}
            </button>
          </div>
        ) : props.tab === 'resources' ? (
          <div className="resource-list">
            {skill.editable && (
              <div className="resource-actions">
                <IconButton
                  icon={Plus}
                  label={t('New resource')}
                  onClick={props.onNewResource}
                />
                <IconButton
                  icon={Upload}
                  label={t('Import resource')}
                  onClick={props.onImportResource}
                />
              </div>
            )}
            {props.resources.map((resource) => {
              const Icon =
                resource.kind === 'link'
                  ? Link
                  : resource.kind === 'image'
                    ? Image
                    : FileText
              return (
                <div className="resource-item" key={resource.path}>
                  <button
                    className={`resource${doc?.path === resource.path ? ' selected' : ''}`}
                    key={resource.path}
                    disabled={resource.kind === 'link'}
                    onClick={() => props.onResource(resource.path)}
                    title={
                      resource.kind === 'link'
                        ? t('Resource links are not followed')
                        : resource.path
                    }
                  >
                    <Icon size={14} />
                    <span>{resource.path}</span>
                    <small>
                      {resource.kind === 'link'
                        ? t('Link')
                        : formatSize(resource.size)}
                    </small>
                  </button>
                  {skill.editable && resource.path !== 'SKILL.md' && (
                    <div className="resource-tools">
                      {resource.kind !== 'link' && (
                        <IconButton
                          icon={FilePenLine}
                          label={t('Rename {name}', { name: resource.path })}
                          onClick={() => props.onRenameResource(resource)}
                        />
                      )}
                      <IconButton
                        icon={Trash2}
                        label={
                          resource.kind === 'link'
                            ? t('Remove link ({path})', { path: resource.path })
                            : t('Move {name} to Trash', { name: resource.path })
                        }
                        onClick={() => props.onTrashResource(resource)}
                      />
                    </div>
                  )}
                </div>
              )
            })}
            {props.resources.length === 0 && (
              <p className="quiet-note">
                {t('No readable resources. Refresh the library to try again.')}
              </p>
            )}
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
              <div className="skill-intro">
                {!translated && !props.raw && !/^\s*#\s/.test(doc.body) && (
                  <>
                    <h1>{skill.name}</h1>
                    {skill.description && <p>{skill.description}</p>}
                  </>
                )}
                {(skill.warnings ?? []).length > 0 && (
                  <details className="parse-warning">
                    <summary>
                      <AlertCircle size={12} />
                      {t('Metadata needs attention')}
                    </summary>
                    {skill.warnings?.map((warning) => (
                      <p key={warning}>{warning}</p>
                    ))}
                  </details>
                )}
              </div>
            )}
            {doc.path !== 'SKILL.md' && (
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
                <Markdown text={translation.text} onLink={props.onLink} />
              ) : (
                <pre className="raw-file">{translation.text}</pre>
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
                <button onClick={props.onReveal}>{t('Show in Finder')}</button>
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
        <span className="path" title={skill.path}>
          {skill.path}
        </span>
        <span className="access-label">
          {doc?.editable ? t('Writable') : t('Read-only')}
        </span>
      </footer>
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
