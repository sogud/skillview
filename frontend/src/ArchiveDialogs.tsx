import { useEffect, useState } from 'react'
import { Archive, ArchiveRestore, Folder, RotateCw } from 'lucide-react'
import * as api from './api'
import type { ArchiveCatalog, ArchivePreview } from './api'
import { IconButton, Modal } from './components'
import { useLocale } from './locales'

export function ArchiveSkillDialog({
  skillId,
  onClose,
  onArchive,
}: {
  skillId: string
  onClose: () => void
  onArchive: (token: string) => Promise<void>
}) {
  const { t, error: message } = useLocale()
  const [preview, setPreview] = useState<ArchivePreview | null>(null)
  const [failure, setFailure] = useState('')
  const [busy, setBusy] = useState(false)
  useEffect(() => {
    let active = true
    void api
      .PlanArchive(skillId)
      .then((value) => {
        if (active) setPreview(value)
      })
      .catch((error) => {
        if (active) setFailure(message(error))
      })
    return () => {
      active = false
    }
  }, [skillId, message])
  async function submit() {
    if (!preview || busy) return
    setBusy(true)
    setFailure('')
    try {
      await onArchive(preview.token)
    } catch (error) {
      setFailure(message(error))
      setPreview(null)
    } finally {
      setBusy(false)
    }
  }
  return (
    <Modal
      title={t('Archive this Skill?')}
      onClose={() => {
        if (!busy) onClose()
      }}
    >
      <div className="modal-body">
        {preview ? (
          <>
            <p>
              {t(
                'Move this copy and all its files to the archive. You can restore it later.',
              )}
            </p>
            <p className="trash-path">{preview.path}</p>
            <p className="quiet-note">{t('Archive folder')}</p>
            <p className="trash-path">{preview.directory}</p>
            {!!preview.links?.length && (
              <p className="form-error">
                {t('These links will point to a missing Skill:')}{' '}
                {preview.links.join(', ')}
              </p>
            )}
            <p className="quiet-note">
              {t(
                'Other copies are kept. Archived Skills are excluded from SkillView scans.',
              )}
            </p>
          </>
        ) : (
          !failure && <p role="status">{t('Preparing archive…')}</p>
        )}
        {failure && (
          <p className="form-error" role="alert">
            {failure}
          </p>
        )}
        <div className="modal-actions">
          <button disabled={busy} onClick={onClose}>
            {t('Cancel')}
          </button>
          <button
            className="primary"
            disabled={busy || !preview}
            onClick={() => void submit()}
          >
            <Archive size={14} /> {t(busy ? 'Archiving…' : 'Archive Skill')}
          </button>
        </div>
      </div>
    </Modal>
  )
}

export function ArchivedSkillsDialog({
  onClose,
  onRestored,
}: {
  onClose: () => void
  onRestored: () => Promise<void>
}) {
  const { t, error: message, language } = useLocale()
  const [catalog, setCatalog] = useState<ArchiveCatalog | null>(null)
  const [failure, setFailure] = useState('')
  const [busy, setBusy] = useState('')
  const [loading, setLoading] = useState(true)
  const [notice, setNotice] = useState('')
  async function refresh() {
    setLoading(true)
    setFailure('')
    try {
      setCatalog(await api.ArchivedSkills())
    } catch (error) {
      setFailure(message(error))
    } finally {
      setLoading(false)
    }
  }
  useEffect(() => {
    let active = true
    void api
      .ArchivedSkills()
      .then((value) => {
        if (active) setCatalog(value)
      })
      .catch((error) => {
        if (active) setFailure(message(error))
      })
      .finally(() => {
        if (active) setLoading(false)
      })
    return () => {
      active = false
    }
  }, [message])
  async function restore(id: string) {
    if (busy) return
    setBusy(id)
    setFailure('')
    setNotice('')
    try {
      await api.RestoreArchivedSkill(id)
      setCatalog((previous) =>
        previous
          ? {
              ...previous,
              entries: (previous.entries ?? []).filter(
                (entry) => entry.id !== id,
              ),
            }
          : previous,
      )
      setNotice(t('Skill restored to its original location.'))
      await onRestored()
      await refresh()
    } catch (error) {
      setFailure(message(error))
    } finally {
      setBusy('')
    }
  }
  return (
    <Modal
      title={t('Archived Skills')}
      onClose={() => {
        if (!busy) onClose()
      }}
    >
      <div className="modal-body archive-browser">
        <div className="archive-location">
          <div>
            <p className="quiet-note">
              {t(
                'Kept until you restore them. Nothing is deleted automatically.',
              )}
            </p>
            <p className="trash-path">{catalog?.directory}</p>
          </div>
          <IconButton
            icon={Folder}
            label={t('Open archive folder')}
            disabled={!catalog?.entries?.length && !catalog?.issues?.length}
            onClick={() =>
              void api
                .RevealArchive()
                .catch((error) => setFailure(message(error)))
            }
          />
          <IconButton
            icon={RotateCw}
            label={t('Refresh archive')}
            disabled={loading || !!busy}
            onClick={() => void refresh()}
          />
        </div>
        {failure && (
          <p className="form-error" role="alert">
            {failure}
          </p>
        )}
        {notice && <p role="status">{notice}</p>}
        {catalog?.issues?.map((issue, index) => (
          <p key={index} className="form-error">
            {t(issue)}
          </p>
        ))}
        {loading && !catalog ? (
          <p role="status">{t('Reading archive…')}</p>
        ) : catalog && !catalog.entries?.length ? (
          <div className="archive-empty">
            <Archive size={24} />
            <p>{t('No archived Skills yet.')}</p>
          </div>
        ) : (
          <ul className="archive-list">
            {catalog?.entries?.map((entry) => (
              <li key={entry.id}>
                <Archive size={16} aria-hidden="true" />
                <div className="archive-details">
                  <strong>{entry.name}</strong>
                  <small>
                    {new Date(entry.archivedAt).toLocaleString(
                      language === 'zh-CN' ? 'zh-CN' : 'en',
                    )}
                  </small>
                  <span className="trash-path" title={entry.originalPath}>
                    {entry.originalPath}
                  </span>
                  {entry.warning && (
                    <p className="form-error">{t(entry.warning)}</p>
                  )}
                </div>
                <button
                  className="archive-restore"
                  disabled={!!busy || !entry.canRestore}
                  aria-label={t('Restore {name}', { name: entry.name })}
                  onClick={() => void restore(entry.id)}
                >
                  <ArchiveRestore size={14} />{' '}
                  {t(busy === entry.id ? 'Restoring…' : 'Restore')}
                </button>
              </li>
            ))}
          </ul>
        )}
        <div className="modal-actions">
          <button disabled={!!busy} onClick={onClose}>
            {t('Close')}
          </button>
        </div>
      </div>
    </Modal>
  )
}
