import { useLocale } from './locales'
import { Virtuoso } from 'react-virtuoso'
import { useEffect, useRef, useState } from 'react'
import { FolderPlus, Link, Folder } from 'lucide-react'
import { Modal, Disclosure } from './components'
import { Select } from './Select'
import * as api from './api'
import type { Source, Catalog, TranslationSettings } from './api'
import { AgentIcon } from './AgentIcon'

export type SettingsTab = 'general' | 'folders' | 'agents' | 'translation'

type SettingsProps = {
  version: string
  onUpdates: () => void
  openDiagnostics: boolean
  tab: SettingsTab
  language: string
  onLanguage: (language: string) => void
  onTranslationSaved: () => void
  theme: string
  accentColor: string
  onAccentColor: (color: string) => void
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
  const [diagnostics, setDiagnostics] = useState<string | null>(null)
  useEffect(() => {
    if (props.openDiagnostics)
      void api
        .UpdateDiagnostics()
        .then(setDiagnostics)
        .catch((failure) => setError(message(failure)))
  }, [props.openDiagnostics, message])
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
  const extraFolders = (props.catalog?.sources ?? []).filter(
    (source) => !source.automatic,
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
          aria-selected={props.tab === 'folders'}
          onClick={() => props.onTab('folders')}
        >
          {t('Folders')}
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
            <h3>{t('Theme color')}</h3>
            <div
              className="accent-palette"
              role="group"
              aria-label={t('Theme color')}
            >
              {[
                { color: '', label: 'Default color' },
                { color: '#3671dc', label: 'Blue' },
                { color: '#8753cb', label: 'Purple' },
                { color: '#278368', label: 'Green' },
                { color: '#bf6639', label: 'Orange' },
                { color: '#c34d6a', label: 'Rose' },
                { color: '#66717e', label: 'Slate' },
              ].map(({ color, label }) => (
                <button
                  key={label}
                  className="accent-swatch"
                  aria-label={t(label)}
                  aria-pressed={props.accentColor === color}
                  title={t(label)}
                  style={
                    { '--swatch': color || '#3671dc' } as React.CSSProperties
                  }
                  onClick={() => props.onAccentColor(color)}
                >
                  {!color ? '↺' : ''}
                </button>
              ))}
              <label className="custom-color">
                {t('Custom color')}
                <input
                  type="color"
                  aria-label={t('Custom color')}
                  value={props.accentColor || '#3671dc'}
                  onChange={(event) => props.onAccentColor(event.target.value)}
                />
              </label>
            </div>
            <label className="field language-field">
              {t('Interface language')}
              <Select
                label={t('Interface language')}
                value={props.language}
                onChange={props.onLanguage}
                options={[
                  { value: 'system', label: t('Follow system') },
                  { value: 'en', label: 'English' },
                  { value: 'zh-CN', label: '简体中文' },
                ]}
              />
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
              <Disclosure className="scan-issues" label={t('Recovery')}>
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
              </Disclosure>
            )}
            <button
              onClick={() => {
                void api
                  .UpdateDiagnostics()
                  .then(setDiagnostics)
                  .catch((failure) => setError(message(failure)))
              }}
            >
              {t('Update diagnostics')}
            </button>
            {diagnostics !== null && (
              <div className="diagnostic-panel">
                <p className="quiet-note">
                  {t(
                    'Saved locally. No Skills, credentials, or private paths are recorded.',
                  )}
                </p>
                <textarea
                  readOnly
                  aria-label={t('Update log')}
                  value={diagnostics}
                />
                <button
                  onClick={() => {
                    void api
                      .RevealUpdateLog()
                      .catch((failure) => setError(message(failure)))
                  }}
                >
                  {t('Show log in Finder')}
                </button>
              </div>
            )}
            <div className="about">
              <Folder size={25} strokeWidth={1.3} />
              <strong>SkillView</strong>
              <span>{t('Version {version}', { version: props.version })}</span>
              <button onClick={props.onUpdates}>
                {t('Check for updates')}
              </button>
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
          <div className="folder-settings">
            <div className="section-heading">
              <h3>{t('Additional folders')}</h3>
            </div>
            <p className="quiet-note">
              {t(
                'Known Agent folders are found automatically. Add a folder here only if it is missing.',
              )}
            </p>
            {extraFolders.length > 0 ? (
              <div
                className="source-list"
                style={{ height: Math.min(235, extraFolders.length * 66) }}
              >
                <Virtuoso
                  data={extraFolders}
                  style={{ height: '100%' }}
                  defaultItemHeight={66}
                  computeItemKey={(_, source) => source.id}
                  itemContent={(_, source) => (
                    <div className="source" key={source.id}>
                      <div className="source-heading">
                        <Folder size={15} />
                        <strong title={source.name}>{t(source.name)}</strong>
                        <span className="source-policy">
                          {source.writable
                            ? t('Editing allowed')
                            : t('Read-only')}
                        </span>
                        <button
                          className="folder-remove"
                          aria-label={t('Remove {name} from SkillView', {
                            name: source.name,
                          })}
                          onClick={() => {
                            void props
                              .onRemove(source.id)
                              .catch((failure) => setError(message(failure)))
                          }}
                        >
                          {t('Remove')}
                        </button>
                      </div>
                      <p className="path" title={source.path}>
                        {source.path}
                      </p>
                      {source.status !== 'ready' && (
                        <p className="source-status attention">
                          {message(source.message)}
                        </p>
                      )}
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
            ) : (
              <div className="folder-empty">
                <Folder size={18} />
                <span>{t('No additional folders')}</span>
              </div>
            )}
            {(props.catalog?.issues?.length ?? 0) > 0 && (
              <Disclosure
                className="scan-issues"
                label={t('{count} paths need attention', {
                  count: props.catalog?.issues?.length ?? 0,
                })}
              >
                {props.catalog?.issues?.map((issue, index) => (
                  <div key={index}>
                    <code>{issue.path}</code>
                    <p>{message(issue.message)}</p>
                    {issue.target && <code>{issue.target}</code>}
                  </div>
                ))}
              </Disclosure>
            )}
            {props.catalog?.discovery?.completed && (
              <Disclosure
                className="scan-issues"
                label={t('{count} directories checked', {
                  count: props.catalog.discovery.directories,
                })}
              >
                {props.catalog.discovery.roots?.map((root) => (
                  <p className="path" key={root} title={root}>
                    {root}
                  </p>
                ))}
              </Disclosure>
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
              <button
                className="folder-add"
                disabled={adding}
                onClick={() => {
                  void add()
                }}
              >
                <FolderPlus size={13} />
                {adding ? t('Choosing folder…') : t('Add folder…')}
              </button>
              <label className="checkbox">
                <input
                  type="checkbox"
                  checked={writable}
                  disabled={adding}
                  aria-label={t('Allow editing in the folder I select')}
                  onChange={(event) => setWritable(event.target.checked)}
                />
                {t('Allow editing')}
              </label>
            </div>
            <p className="quiet-note">
              {t('Added folders are read-only unless you allow editing.')}
            </p>
            {error && (
              <p className="form-error" role="alert">
                {error}
              </p>
            )}
            {extraFolders.length > 0 && (
              <p className="quiet-note">
                {t(
                  'Removing a source only removes it from this list. Its files stay in place.',
                )}
              </p>
            )}
          </div>
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
              <Select
                label={t('Source')}
                value={selectedSource?.id ?? ''}
                onChange={setSourceId}
                options={writable.map((source) => ({
                  value: source.id,
                  label: source.name,
                  description: source.resolvedPath,
                }))}
              />
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
  const [providerKey, setProviderKey] = useState('')
  const [clearAI, setClearAI] = useState(false)
  const [clearProvider, setClearProvider] = useState(false)
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
        '',
        clearAI,
        false,
        providerKey,
        clearProvider,
      )
      setAIKey('')
      setProviderKey('')
      setClearAI(false)
      setClearProvider(false)
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
          <Select
            label={t('Translation provider')}
            value={settings.provider}
            onChange={(provider) => {
              setSettings({ ...settings, provider })
              setProviderKey('')
              setClearProvider(false)
              setConnectionOK(false)
              setSaved(false)
            }}
            options={[
              {
                value: 'google',
                label: t('Google Translate (free web service)'),
              },
              {
                value: 'youdao',
                label: t('Youdao Translate (free web service)'),
              },
              {
                value: 'libretranslate',
                label: t('LibreTranslate (self-hosted)'),
              },
              { value: 'ai', label: t('AI translation') },
            ]}
          />
        </label>
        <label className="field">
          {t('Target language')}
          <Select
            label={t('Target language')}
            value={settings.targetLanguage}
            onChange={(targetLanguage) => {
              setSettings({ ...settings, targetLanguage })
              setSaved(false)
            }}
            options={targetLanguages.map(([value, label]) => ({
              value,
              label,
            }))}
          />
        </label>
        {settings.provider === 'youdao' && (
          <p className="quiet-note">
            {t(
              'Youdao uses its free web translation service. No API key is needed.',
            )}
          </p>
        )}
        {settings.provider === 'google' && (
          <p className="quiet-note">
            {t(
              "Google's public web service can be rate limited. You can also choose Youdao or a local AI model.",
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
        {settings.provider === 'libretranslate' && (
          <>
            <label className="field">
              {t('LibreTranslate server URL')}
              <input
                type="url"
                required
                value={settings.libreUrl}
                onChange={(event) =>
                  setSettings({ ...settings, libreUrl: event.target.value })
                }
                placeholder="http://localhost:5000"
              />
            </label>
            <p className="quiet-note">
              {t(
                'Connect your own LibreTranslate server. Available languages depend on its installed models.',
              )}
            </p>
            <label className="field">
              {t('API key')}
              <input
                type="password"
                autoComplete="off"
                value={providerKey}
                onChange={(event) => setProviderKey(event.target.value)}
                placeholder={t(
                  settings.hasLibreKey
                    ? 'Leave blank to keep the saved key'
                    : 'Optional for your server',
                )}
              />
            </label>
            {settings.hasLibreKey && (
              <label className="checkbox">
                <input
                  type="checkbox"
                  checked={clearProvider}
                  onChange={(event) => setClearProvider(event.target.checked)}
                />
                {t('Remove saved key')}
              </label>
            )}
          </>
        )}
        {(settings.hasAIKey || settings.hasLibreKey) && (
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
