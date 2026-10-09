import { Download, RotateCw } from 'lucide-react'
import type { UpdateInfo } from './api'
import { Markdown, Modal } from './components'
import { useLocale } from './locales'

export function UpdateDialog({
  status,
  onCheck,
  onDownload,
  onRestart,
  onLink,
  onClose,
  onDiagnostics,
}: {
  status: UpdateInfo | null
  onCheck: () => void
  onDownload: () => void
  onRestart: () => void
  onLink: (url: string) => void
  onClose: () => void
  onDiagnostics: () => void
}) {
  const { t } = useLocale()
  const checking = !status || status.state === 'checking'
  const downloading = status?.state === 'downloading'
  const verifying =
    status?.state === 'verifying' || status?.state === 'extracting'
  const busy = downloading || verifying
  const ready = status?.state === 'ready'
  const available = !!status?.latest
  const progress = status?.total
    ? Math.min(100, Math.round((status.written / status.total) * 100))
    : null
  return (
    <Modal title={t('SkillView updates')} onClose={onClose}>
      <div className="modal-body update-body">
        <p className="quiet-note">
          {t('Version {version}', { version: status?.version ?? '…' })}
        </p>
        <h3 role="status">
          {checking
            ? t('Checking for updates…')
            : ready
              ? t('Ready to update')
              : verifying
                ? t('Verifying update…')
                : downloading
                  ? t('Downloading update…')
                  : available
                    ? t('Version {version} is available', {
                        version: status.latest,
                      })
                    : status?.state === 'up-to-date'
                      ? t('You’re up to date')
                      : t('Check for a new version')}
        </h3>
        {status?.message && (
          <p
            role={status.state === 'error' ? 'alert' : 'status'}
            className={status.state === 'error' ? 'form-error' : 'quiet-note'}
          >
            {t(status.message)}
          </p>
        )}
        {busy && (
          <div className="update-progress">
            <progress
              aria-label={t('Download progress')}
              max={100}
              {...(progress === null ? {} : { value: progress })}
            />
            <span>
              {verifying
                ? t('Verifying update…')
                : progress === null
                  ? t('Downloading…')
                  : `${progress}%`}
            </span>
          </div>
        )}
        {ready && (
          <p className="quiet-note">
            {t('Restart to finish updating. You can save any changes first.')}
          </p>
        )}
        {!!status?.notes && (
          <section className="update-notes" aria-label={t('Release notes')}>
            <Markdown text={status.notes} onLink={onLink} />
          </section>
        )}
        <p className="quiet-note">
          {t(
            'SkillView checks GitHub Releases when it opens. Downloads start only when you choose to update.',
          )}
        </p>
      </div>
      <div className="modal-actions">
        <button onClick={onDiagnostics}>{t('Update diagnostics')}</button>
        <button
          onClick={() =>
            onLink(
              status?.releaseURL ||
                'https://github.com/sogud/skillview/releases',
            )
          }
        >
          {t('View on GitHub')}
        </button>
        {ready ? (
          <button className="primary" onClick={onRestart}>
            {t('Restart to update')}
          </button>
        ) : available && status.canInstall && !checking ? (
          <button className="primary" disabled={busy} onClick={onDownload}>
            <Download size={14} />
            {t(
              verifying
                ? 'Verifying update…'
                : downloading
                  ? 'Downloading…'
                  : 'Download update',
            )}
          </button>
        ) : (
          <button disabled={checking || busy} onClick={onCheck}>
            <RotateCw size={14} />
            {t(checking ? 'Checking…' : 'Check for updates')}
          </button>
        )}
      </div>
    </Modal>
  )
}
