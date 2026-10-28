import { useEffect, useState } from 'react'
import * as api from './api'
import { Modal } from './components'
import { useLocale } from './locales'

type Preview = Awaited<ReturnType<typeof api.PreviewFeedback>>

export function FeedbackDialog({
  initialIncludeAIError,
  onClose,
}: {
  initialIncludeAIError: boolean
  onClose: () => void
}) {
  const { t, error: errorMessage } = useLocale()
  const [message, setMessage] = useState('')
  const [includeAIError, setIncludeAIError] = useState(false)
  const [hasAIError, setHasAIError] = useState(false)
  const [ready, setReady] = useState(false)
  const [preview, setPreview] = useState<Preview | null>(null)
  const [destination, setDestination] = useState('')
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState('')
  const [receipt, setReceipt] = useState('')

  useEffect(() => {
    let active = true
    void api
      .FeedbackConfiguration()
      .then((config) => {
        if (!active) return
        if (config.enabled) setDestination(config.destination)
        setHasAIError(Boolean(config.aiError))
        setIncludeAIError(initialIncludeAIError && Boolean(config.aiError))
        setReady(true)
      })
      .catch((failure) => {
        if (active) setError(errorMessage(failure))
      })
    return () => {
      active = false
    }
  }, [])

  const review = async () => {
    setBusy(true)
    setError('')
    try {
      setPreview(
        await api.PreviewFeedback({
          message,
          includeAIError,
        }),
      )
    } catch (failure) {
      setError(errorMessage(failure))
    } finally {
      setBusy(false)
    }
  }
  const send = async () => {
    if (!preview) return
    setBusy(true)
    setError('')
    try {
      const result = await api.SendFeedback(preview)
      setReceipt(result.event_id)
    } catch (failure) {
      setError(errorMessage(failure))
    } finally {
      setBusy(false)
    }
  }

  return (
    <Modal
      title={t('Send feedback')}
      onClose={() => {
        if (!busy) onClose()
      }}
    >
      <div className="modal-body feedback-body">
        <p className="quiet-note">
          {t(
            'Only your reviewed report is sent. Documents, model responses, screenshots and logs are not attached. Do not paste private content or secrets.',
          )}
        </p>
        {!destination && (
          <p className="quiet-note">
            {t('Feedback is not configured in this build.')}
          </p>
        )}
        {receipt ? (
          <p role="status">{t('Report received: {id}', { id: receipt })}</p>
        ) : (
          <>
            <label htmlFor="feedback-message">
              {t('Describe the problem')}
            </label>
            <textarea
              id="feedback-message"
              dir="auto"
              data-autofocus
              rows={5}
              maxLength={4000}
              value={message}
              disabled={busy}
              onChange={(event) => {
                setMessage(event.target.value)
                setPreview(null)
                setError('')
              }}
            />
            {hasAIError && (
              <label className="feedback-toggle">
                <input
                  type="checkbox"
                  checked={includeAIError}
                  disabled={busy}
                  onChange={(event) => {
                    setIncludeAIError(event.target.checked)
                    setPreview(null)
                    setError('')
                  }}
                />
                {t(
                  'Include the latest AI response format diagnostic (no content)',
                )}
              </label>
            )}
            {preview && (
              <div>
                <h3>{t('Report preview')}</h3>
                <p className="quiet-note">
                  {destination || t('Local preview only')}
                </p>
                <pre className="feedback-preview">
                  {JSON.stringify(preview, null, 2)}
                </pre>
              </div>
            )}
            {error && (
              <p className="form-error" role="alert">
                {error}
              </p>
            )}
          </>
        )}
      </div>
      <div className="modal-actions">
        <button disabled={busy} onClick={onClose}>
          {t('Close')}
        </button>
        {!receipt && (
          <button
            disabled={busy || !ready || !message.trim()}
            onClick={() => void review()}
          >
            {t('Review report')}
          </button>
        )}
        {!receipt && preview && (
          <button
            className="primary"
            disabled={busy || !destination}
            onClick={() => void send()}
          >
            {t(
              busy
                ? 'Sending…'
                : error
                  ? 'Retry report'
                  : 'Send reviewed report',
            )}
          </button>
        )}
      </div>
    </Modal>
  )
}
