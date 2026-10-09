import { useLocale } from './locales'
import { Virtuoso } from 'react-virtuoso'
import { useEffect, useRef, useState } from 'react'
import { FolderPlus, RotateCw, Link, Folder, X } from 'lucide-react'
import { IconButton, Modal } from './components'
import * as api from './api'
import type { Source, Catalog, TranslationSettings } from './api'
import { AgentIcon } from './AgentIcon'

export type SettingsTab = 'general' | 'sources' | 'agents' | 'translation'

type SettingsProps = {
  tab: SettingsTab
  language: string
  onLanguage: (language: string) => void
  onTranslationSaved: () => void
  theme: string
  catalog: Catalog | null
  onTab: (tab: SettingsTab) => void
  onTheme: (theme: string) => void
  onAdd: (writable: boolean) => Promise<void>
  onRemove: (id: string) => Promise<void>
  onRescan: () => void
  onClose: () => void
}

export function Settings(props: SettingsProps) {
  const { t, error: message } = useLocale()
  const [recentTrash, setRecentTrash] = useState<api.TrashReceipt[]>([])
  const [cacheMessage, setCacheMessage] = useState('')
  useEffect(() => {
    let active = true
    void api
      .Preferences()
      .then((value) => {
        if (active) setRecentTrash(value.recentTrash ?? [])
      })
      .catch(() => {})
    return () => {
      active = false
    }
  }, [])
  const [showAll, setShowAll] = useState(false)
  const agents = (props.catalog?.agents ?? []).filter(
    (agent) => showAll || agent.installed,
  )
  const [writable, setWritable] = useState(false)
  const [adding, setAdding] = useState(false)
  const [error, setError] = useState('')
  async function add() {
    setAdding(true)
    setError('')
    try {
      await props.onAdd(writable)
    } catch (failure) {
      setError(message(failure))
    } finally {
      setAdding(false)
    }
  }
  return (
    <Modal title={t('Settings')} onClose={props.onClose}>
      <div
        className="modal-tabs"
        role="tablist"
        aria-label={t('Settings sections')}
      >
        <button
          role="tab"
          aria-selected={props.tab === 'general'}
          onClick={() => props.onTab('general')}
        >
          {t('General')}
        </button>
        <button
          role="tab"
          aria-selected={props.tab === 'sources'}
          onClick={() => props.onTab('sources')}
        >
          {t('Sources')}
        </button>
        <button
          role="tab"
          aria-selected={props.tab === 'agents'}
          onClick={() => props.onTab('agents')}
        >
          Agents
        </button>
        <button
          role="tab"
          aria-selected={props.tab === 'translation'}
          onClick={() => props.onTab('translation')}
        >
          {t('Translation')}
        </button>
      </div>
      <div className="modal-body">
        {props.tab === 'general' ? (
          <>
            <h3>{t('Appearance')}</h3>
            <div
              className="segmented"
              role="group"
              aria-label={t('Appearance')}
            >
              {['system', 'light', 'dark'].map((theme) => (
                <button
                  key={theme}
                  aria-pressed={props.theme === theme}
                  onClick={() => props.onTheme(theme)}
                >
                  {t(theme[0].toUpperCase() + theme.slice(1))}
                </button>
              ))}
            </div>
            <label className="field language-field">
              {t('Interface language')}
              <select
                value={props.language}
                onChange={(event) => props.onLanguage(event.target.value)}
              >
                <option value="system">{t('Follow system')}</option>
                <option value="en">English</option>
                <option value="zh-CN">简体中文</option>
              </select>
            </label>
            <button
              onClick={() => {
                void api
                  .ClearCache()
                  .then(() => {
                    setCacheMessage(t('Cache cleared'))
                    props.onTranslationSaved()
                  })
                  .catch((failure) => setError(message(failure)))
              }}
            >
              {t('Clear cache')}
            </button>
            {cacheMessage && (
              <p className="quiet-note" role="status">
                {cacheMessage}
              </p>
            )}
            {error && (
              <p className="form-error" role="alert">
                {error}
              </p>
            )}
            {!!recentTrash.length && (
              <details className="scan-issues">
                <summary>{t('Recovery')}</summary>
                {recentTrash
                  .slice(-5)
                  .reverse()
                  .map((receipt) => (
                    <div key={receipt.id}>
                      <p className="path" title={receipt.originalPath}>
                        {receipt.originalPath}
                      </p>
                      <button
                        onClick={() => {
                          void api
                            .UndoTrash(receipt.id)
                            .then(() => {
                              setRecentTrash((items) =>
                                items.filter((item) => item.id !== receipt.id),
                              )
                              props.onRescan()
                            })
                            .catch((failure) => setError(message(failure)))
                        }}
                      >
                        {t('Restore {name}', {
                          name: receipt.originalPath.split('/').pop() ?? '',
                        })}
                      </button>
                    </div>
                  ))}
              </details>
            )}
            <div className="about">
              <Folder size={25} strokeWidth={1.3} />
              <strong>SkillView</strong>
              <span>{t('Version {version}', { version: '0.1.0' })}</span>
              <p>
                {t('Read, browse, and edit your local Skills.')}
                <br />
                {t('Your files stay in their current folders.')}
              </p>
            </div>
          </>
        ) : props.tab === 'translation' ? (
          <TranslationSettingsForm onSaved={props.onTranslationSaved} />
        ) : props.tab === 'agents' ? (
          <>
            <label className="checkbox">
              <input
                type="checkbox"
                checked={showAll}
                onChange={(event) => setShowAll(event.target.checked)}
              />
              {t('Show all supported Agents')}
            </label>
            <p className="quiet-note">
              {t(
                'Default view shows installed Agents. Configuration-only entries are marked separately.',
              )}
            </p>
            <div className="agent-list">
              {agents.map((agent) => (
                <div className="agent-entry" key={agent.id}>
                  <AgentIcon icon={agent.icon} name={agent.name} size={20} />
                  <div>
                    <strong>{agent.name}</strong>
                    <span className="path" title={agent.evidence}>
                      {agent.evidence}
                    </span>
                  </div>
                  <small>
                    {t(
                      agent.installed
                        ? 'Installed'
                        : agent.configured
                          ? 'Configuration found'
                          : 'Not detected',
                    )}
                  </small>
                </div>
              ))}
              {!agents.length && (
                <p className="quiet-note">
                  {t(
                    'No installed Agents detected. Skills folders can still be added.',
                  )}
                </p>
              )}
            </div>
          </>
        ) : (
          <>
            <div className="section-heading">
              <h3>{t('Skills folders')}</h3>
              <IconButton
                icon={RotateCw}
                label={t('Rescan folders')}
                onClick={props.onRescan}
              />
            </div>
            <div className="source-list">
              <Virtuoso
                data={props.catalog?.sources ?? []}
                style={{ height: 235 }}
                defaultItemHeight={90}
                computeItemKey={(_, source) => source.id}
                itemContent={(_, source) => (
                  <div className="source" key={source.id}>
                    <div className="source-heading">
                      <Folder size={15} />
                      <strong>{t(source.name)}</strong>
                      <span className="source-policy">
                        {source.writable
                          ? t('Editing allowed')
                          : t('Read-only')}
                      </span>
                      <IconButton
                        icon={X}
                        label={t('Remove {name} from SkillView', {
                          name: source.name,
                        })}
                        onClick={() => {
                          void props
                            .onRemove(source.id)
                            .catch((failure) => setError(message(failure)))
                        }}
                      />
                    </div>
                    <p className="path" title={source.path}>
                      {source.path}
                    </p>
                    <p
                      className={`source-status ${source.status === 'ready' ? '' : 'attention'}`}
                    >
                      {source.status === 'ready'
                        ? t('Found for {tools}', {
                            tools: (source.tools ?? []).join(', '),
                          })
                        : message(source.message)}
                    </p>
                    {source.resolvedPath &&
                      source.resolvedPath !== source.path && (
                        <p className="link-target">
                          <Link size={12} />
                          <span title={source.resolvedPath}>
                            {source.resolvedPath}
                          </span>
                        </p>
                      )}
                  </div>
                )}
              />
            </div>
            {(props.catalog?.issues?.length ?? 0) > 0 && (
              <details className="scan-issues">
                <summary>
                  {t('{count} paths need attention', {
                    count: props.catalog?.issues?.length ?? 0,
                  })}
                </summary>
                {props.catalog?.issues?.map((issue, index) => (
                  <div key={index}>
                    <code>{issue.path}</code>
                    <p>{message(issue.message)}</p>
                    {issue.target && <code>{issue.target}</code>}
                  </div>
                ))}
              </details>
            )}
            {props.catalog?.discovery?.completed && (
              <details className="scan-issues">
                <summary>
                  {t('{count} directories checked', {
                    count: props.catalog.discovery.directories,
                  })}
                </summary>
                {props.catalog.discovery.roots?.map((root) => (
                  <p className="path" key={root} title={root}>
                    {root}
                  </p>
                ))}
              </details>
            )}
            {props.catalog?.discovery?.limited && (
              <p className="quiet-note">
                {t(
                  'Search limit reached. Add a narrower folder to search the remaining locations.',
                )}
              </p>
            )}
            {!!props.catalog?.discovery?.denied && (
              <p className="quiet-note">
                {t(
                  'Some folders could not be read. Add them explicitly if needed.',
                )}
              </p>
            )}
            <div className="add-source">
              <label className="checkbox">
                <input
                  type="checkbox"
                  checked={writable}
                  onChange={(event) => setWritable(event.target.checked)}
                />
                {t('Allow editing in the folder I select')}
              </label>
              <button
                className="primary"
                disabled={adding}
                onClick={() => {
                  void add()
                }}
              >
                <FolderPlus size={14} />
                {adding ? t('Choosing folder…') : t('Add Skills folder')}
              </button>
            </div>
            {error && (
              <p className="form-error" role="alert">
                {error}
              </p>
            )}
            <p className="quiet-note">
              {t(
                'Removing a source only removes it from this list. Its files stay in place.',
              )}
            </p>
          </>
        )}
      </div>
    </Modal>
  )
}

export function CreateDialog({
  sources,
  onCreate,
  onAddSource,
  onClose,
}: {
  sources: Source[]
  onCreate: (
    sourceId: string,
    name: string,
    description: string,
  ) => Promise<void>
  onAddSource: () => Promise<void>
  onClose: () => void
}) {
  const { t, error: message } = useLocale()
  const writable = sources.filter(
    (source) => source.writable && source.status === 'ready',
  )
  const [sourceId, setSourceId] = useState(writable[0]?.id ?? '')
  const [name, setName] = useState('')
  const [description, setDescription] = useState('')
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState('')
  const selectedSource =
    writable.find((source) => source.id === sourceId) ?? writable[0]
  async function create() {
    if (!selectedSource) return
    setBusy(true)
    setError('')
    try {
      await onCreate(selectedSource.id, name, description)
    } catch (failure) {
      setError(message(failure))
    } finally {
      setBusy(false)
    }
  }
  return (
    <Modal
      title={t('New Skill')}
      onClose={() => {
        if (!busy) onClose()
      }}
    >
      <form
        className="modal-body"
        onSubmit={(event) => {
          event.preventDefault()
          void create()
        }}
      >
        {writable.length === 0 ? (
          <div className="empty-dialog">
            <FolderPlus size={28} />
            <h3>{t('Choose a folder for editing')}</h3>
            <p>{t('New Skills need a source where editing is allowed.')}</p>
            <button
              type="button"
              className="primary"
              onClick={() => {
                void onAddSource().catch((failure) =>
                  setError(message(failure)),
                )
              }}
            >
              {t('Choose editable folder')}
            </button>
            {error && (
              <p className="form-error" role="alert">
                {error}
              </p>
            )}
          </div>
        ) : (
          <>
            <label className="field">
              {t('Source')}
              <select
                value={selectedSource?.id ?? ''}
                onChange={(event) => setSourceId(event.target.value)}
              >
                {writable.map((source) => (
                  <option key={source.id} value={source.id}>
                    {source.name}
                  </option>
                ))}
              </select>
            </label>
            <label className="field">
              {t('Name')}
              <input
                data-autofocus
                value={name}
                onChange={(event) => setName(event.target.value)}
                placeholder="my-skill"
                required
                pattern="[a-z0-9]+(-[a-z0-9]+)*"
                maxLength={64}
              />
            </label>
            <label className="field">
              {t('Description')}
              <textarea
                value={description}
                onChange={(event) => setDescription(event.target.value)}
                placeholder={t('What it does and when to use it.')}
                required
                maxLength={1024}
                rows={3}
              />
            </label>
            <p className="quiet-note path" title={selectedSource?.resolvedPath}>
              {selectedSource?.resolvedPath}/{name || 'my-skill'}/SKILL.md
            </p>
            {error && (
              <p className="form-error" role="alert">
                {error}
              </p>
            )}
            <div className="modal-actions">
              <button type="button" onClick={onClose} disabled={busy}>
                {t('Cancel')}
              </button>
              <button
                className="primary"
                disabled={busy || !name || !description.trim()}
              >
                {busy ? t('Creating…') : t('Create Skill')}
              </button>
            </div>
          </>
        )}
      </form>
    </Modal>
  )
}

const targetLanguages = [
  ['zh-CN', '简体中文'],
  ['zh-TW', '繁體中文'],
  ['en', 'English'],
  ['ja', '日本語'],
  ['ko', '한국어'],
  ['fr', 'Français'],
  ['de', 'Deutsch'],
  ['es', 'Español'],
  ['pt', 'Português'],
  ['ru', 'Русский'],
]

function TranslationSettingsForm({ onSaved }: { onSaved: () => void }) {
  const { t, error: message } = useLocale()
  const [settings, setSettings] = useState<TranslationSettings | null>(null)
  const [aiKey, setAIKey] = useState('')
  const [googleKey, setGoogleKey] = useState('')
  const [clearAI, setClearAI] = useState(false)
  const [clearGoogle, setClearGoogle] = useState(false)
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState('')
  const [saved, setSaved] = useState(false)
  const [testing, setTesting] = useState(false)
  const [connectionOK, setConnectionOK] = useState(false)
  const connectionRequest = useRef<ReturnType<
    typeof api.TestTranslationConnection
  > | null>(null)
  useEffect(
    () => () => {
      connectionRequest.current?.cancel()
    },
    [],
  )
  useEffect(() => {
    let active = true
    void api
      .GetTranslationSettings()
      .then((value) => {
        if (active) setSettings(value)
      })
      .catch((failure) => {
        if (active) setError(message(failure))
      })
    return () => {
      active = false
    }
  }, [message])
  async function save(testConnection = false) {
    if (!settings) return
    setBusy(true)
    setError('')
    setSaved(false)
    setConnectionOK(false)
    try {
      await api.SetTranslationSettings(
        settings,
        aiKey,
        googleKey,
        clearAI,
        clearGoogle,
      )
      setAIKey('')
      setGoogleKey('')
      setClearAI(false)
      setClearGoogle(false)
      setSettings(await api.GetTranslationSettings())
      setSaved(true)
      onSaved()
      if (testConnection) {
        setTesting(true)
        const request = api.TestTranslationConnection()
        connectionRequest.current = request
        await request
        setConnectionOK(true)
      }
    } catch (failure) {
      setError(message(failure))
    } finally {
      setBusy(false)
      setTesting(false)
      connectionRequest.current = null
    }
  }
  if (!settings)
    return (
      <p
        className={error ? 'form-error' : 'quiet-note'}
        role={error ? 'alert' : 'status'}
      >
        {error || t('Loading settings…')}
      </p>
    )
  return (
    <form
      onSubmit={(event) => {
        event.preventDefault()
        void save()
      }}
    >
      <fieldset disabled={busy} className="translation-fields">
        <label className="field">
          {t('Translation provider')}
          <select
            value={settings.provider}
            onChange={(event) => {
              setSettings({ ...settings, provider: event.target.value })
              setSaved(false)
            }}
          >
            <option value="google">
              {t('Google Translate (free web service)')}
            </option>
            <option value="google-cloud">
              {t('Google Cloud Translation')}
            </option>
            <option value="ai">{t('AI translation')}</option>
          </select>
        </label>
        <label className="field">
          {t('Target language')}
          <select
            value={settings.targetLanguage}
            onChange={(event) =>
              setSettings({ ...settings, targetLanguage: event.target.value })
            }
          >
            {targetLanguages.map(([id, label]) => (
              <option key={id} value={id}>
                {label}
              </option>
            ))}
          </select>
        </label>
        {settings.provider === 'google' && (
          <p className="quiet-note">
            {t(
              "Google's public web service can be rate limited. Use Google Cloud for an official API.",
            )}
          </p>
        )}
        {settings.provider === 'ai' && (
          <>
            <label className="field">
              {t('Base URL')}
              <input
                type="url"
                required
                value={settings.baseUrl}
                onChange={(event) =>
                  setSettings({ ...settings, baseUrl: event.target.value })
                }
                placeholder="http://localhost:11434/v1"
              />
            </label>
            <label className="field">
              {t('Model')}
              <input
                required
                value={settings.model}
                onChange={(event) =>
                  setSettings({ ...settings, model: event.target.value })
                }
                placeholder="qwen3:8b"
              />
            </label>
            <label className="field">
              {t('API key')}
              <input
                type="password"
                autoComplete="off"
                value={aiKey}
                onChange={(event) => setAIKey(event.target.value)}
                placeholder={t(
                  settings.hasAIKey
                    ? 'Leave blank to keep the saved key'
                    : 'Optional for a local model',
                )}
              />
            </label>
            {settings.hasAIKey && (
              <label className="checkbox">
                <input
                  type="checkbox"
                  checked={clearAI}
                  onChange={(event) => setClearAI(event.target.checked)}
                />
                {t('Remove saved key')}
              </label>
            )}
            <p className="quiet-note">
              {t(
                'Connect a local model through its OpenAI-compatible URL. No API key is needed for most local servers.',
              )}
            </p>
          </>
        )}
        {settings.provider === 'google-cloud' && (
          <>
            <label className="field">
              {t('Google Cloud API key')}
              <input
                type="password"
                autoComplete="off"
                value={googleKey}
                onChange={(event) => setGoogleKey(event.target.value)}
                placeholder={t(
                  settings.hasGoogleKey
                    ? 'Leave blank to keep the saved key'
                    : 'API key',
                )}
              />
            </label>
            {settings.hasGoogleKey && (
              <label className="checkbox">
                <input
                  type="checkbox"
                  checked={clearGoogle}
                  onChange={(event) => setClearGoogle(event.target.checked)}
                />
                {t('Remove saved key')}
              </label>
            )}
          </>
        )}
        {(settings.hasAIKey || settings.hasGoogleKey) && (
          <p className="quiet-note">{t('Key saved in macOS Keychain')}</p>
        )}
        <p className="quiet-note">
          {t(
            'Cloud providers receive the displayed document only when you click Translate. Originals are never changed.',
          )}
        </p>
        {error && (
          <p className="form-error" role="alert">
            {error}
          </p>
        )}
        {saved && (
          <p className="quiet-note" role="status">
            {t('Settings saved')}
          </p>
        )}
        {connectionOK && (
          <p className="quiet-note" role="status">
            {t('Connection test passed')}
          </p>
        )}
        <p className="quiet-note">
          {t(
            'Connection testing sends only “Hello.” to the selected service and may use provider credits.',
          )}
        </p>
        <div className="modal-actions">
          <button
            type="button"
            onClick={() => {
              void save(true)
            }}
            disabled={busy}
          >
            {t(testing ? 'Testing connection…' : 'Save and test connection')}
          </button>
          <button className="primary" disabled={busy}>
            {t(busy ? 'Saving…' : 'Save settings')}
          </button>
        </div>
      </fieldset>
    </form>
  )
}

export function ResourceDialog({
  kind,
  resource,
  skillId,
  onSubmit,
  onClose,
}: {
  kind: 'new' | 'rename'
  resource?: api.Resource
  skillId: string
  onSubmit: (path: string, text: string) => Promise<void>
  onClose: () => void
}) {
  const { t, error: message } = useLocale()
  const [path, setPath] = useState(resource?.path ?? 'notes.md')
  const [text, setText] = useState('')
  const [references, setReferences] = useState<string[]>([])
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState('')
  useEffect(() => {
    let active = true
    if (resource)
      void api
        .ResourceReferences(skillId, resource.path)
        .then((items) => {
          if (active) setReferences(items ?? [])
        })
        .catch((failure) => {
          if (active) setError(message(failure))
        })
    return () => {
      active = false
    }
  }, [resource, skillId, message])
  async function submit() {
    setBusy(true)
    setError('')
    try {
      await onSubmit(path, text)
    } catch (failure) {
      setError(message(failure))
    } finally {
      setBusy(false)
    }
  }
  return (
    <Modal
      title={t(kind === 'new' ? 'New resource' : 'Rename resource')}
      onClose={() => {
        if (!busy) onClose()
      }}
    >
      <form
        className="modal-body"
        onSubmit={(event) => {
          event.preventDefault()
          void submit()
        }}
      >
        <label className="field">
          {t('File path')}
          <input
            data-autofocus
            required
            value={path}
            onChange={(event) => setPath(event.target.value)}
            placeholder="references/notes.md"
            disabled={busy}
          />
        </label>
        {kind === 'new' && (
          <label className="field">
            {t('Text content')}
            <textarea
              rows={5}
              value={text}
              onChange={(event) => setText(event.target.value)}
              disabled={busy}
            />
          </label>
        )}
        {!!references.length && (
          <p className="quiet-note">
            {t('Update the local links in these files after renaming:')}{' '}
            {references.join(', ')}
          </p>
        )}
        {error && (
          <p className="form-error" role="alert">
            {error}
          </p>
        )}
        <div className="modal-actions">
          <button type="button" onClick={onClose} disabled={busy}>
            {t('Cancel')}
          </button>
          <button className="primary" disabled={busy || !path.trim()}>
            {t(
              busy ? 'Saving…' : kind === 'new' ? 'Create resource' : 'Rename',
            )}
          </button>
        </div>
      </form>
    </Modal>
  )
}

export function TrashDialog({
  skillId,
  path,
  link,
  onTrash,
  onClose,
}: {
  skillId: string
  path: string
  link?: { skillId: string; sourceId: string; path: string }
  onTrash: (token: string) => Promise<void>
  onClose: () => void
}) {
  const { t, error: message } = useLocale()
  const [preview, setPreview] = useState<api.TrashPreview | null>(null)
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState('')
  useEffect(() => {
    let active = true
    const request = link
      ? api.PlanLinkTrash(link.skillId, link.sourceId, link.path)
      : api.PlanTrash(skillId, path)
    void request
      .then((value) => {
        if (active) setPreview(value)
      })
      .catch((failure) => {
        if (active) setError(message(failure))
      })
    return () => {
      active = false
    }
  }, [skillId, path, link, message])
  async function submit() {
    if (!preview) return
    setBusy(true)
    setError('')
    try {
      await onTrash(preview.token)
    } catch (failure) {
      setError(message(failure))
      setPreview(null)
    } finally {
      setBusy(false)
    }
  }
  const isLink = !!link || !!preview?.link
  return (
    <Modal
      title={t(
        isLink
          ? 'Remove this link?'
          : path
            ? 'Move resource to Trash?'
            : 'Move Skill to Trash?',
      )}
      onClose={() => {
        if (!busy) onClose()
      }}
    >
      <div className="modal-body">
        {preview && (
          <>
            <p className="trash-path">{preview.path}</p>
            {isLink ? (
              <p>
                {t(
                  'Only this link will be moved to Trash. The original file or folder is kept.',
                )}
              </p>
            ) : (
              <p>
                {t('{count} files will be moved to macOS Trash.', {
                  count: preview.count,
                })}
              </p>
            )}
            {preview.linkTarget && (
              <p className="quiet-note trash-path">
                {t('Link target')}: {preview.linkTarget}
              </p>
            )}
            {!!preview.links?.length && (
              <p className="quiet-note">
                {t('These links will point to a missing Skill:')}{' '}
                {preview.links.join(', ')}
              </p>
            )}
            <p className="quiet-note">
              {t('Use Undo after the move to restore the original location.')}
            </p>
          </>
        )}
        {error && (
          <p className="form-error" role="alert">
            {error}
          </p>
        )}
        <div className="modal-actions">
          <button onClick={onClose} disabled={busy}>
            {t('Cancel')}
          </button>
          <button
            className="danger"
            onClick={() => {
              void submit()
            }}
            disabled={busy || !preview}
          >
            {t(busy ? 'Moving…' : isLink ? 'Remove link' : 'Move to Trash')}
          </button>
        </div>
      </div>
    </Modal>
  )
}
