import {
  useEffect,
  useImperativeHandle,
  useRef,
  useState,
  type Ref,
  type CSSProperties,
} from 'react'
import {
  ArrowLeft,
  Package,
  Search,
  RotateCw,
  FileCode,
  Eye,
  Lock,
  LoaderCircle,
} from 'lucide-react'
import * as api from './api'
import type { Plugin, PluginCatalog, Document, Resource } from './api'
import { AgentIcon } from './AgentIcon'
import { FileTree } from './FileTree'
import { IconButton, Markdown } from './components'
import { useLocale } from './locales'

export type PluginsHandle = { command: (name: string) => void }

export function Plugins({
  active,
  fontSize,
  explorerWidth,
  ref,
}: {
  active: boolean
  fontSize: number
  explorerWidth: number
  ref: Ref<PluginsHandle>
}) {
  const { t, error: message } = useLocale()
  const [catalog, setCatalog] = useState<PluginCatalog | null>(null)
  const [loading, setLoading] = useState(false)
  const [failure, setFailure] = useState('')
  const [selected, setSelected] = useState<Plugin | null>(null)
  const [query, setQuery] = useState('')
  const [agent, setAgent] = useState('')
  const search = useRef<HTMLInputElement>(null)
  const focusSearch = useRef(false)
  const sequence = useRef(0)

  async function refresh() {
    const request = ++sequence.current
    setLoading(true)
    setFailure('')
    try {
      const next = await api.Plugins()
      if (request !== sequence.current) return
      setCatalog(next)
      setAgent((previous) =>
        (next.plugins ?? []).some((p) => p.agent === previous) ? previous : '',
      )
      setSelected((previous) =>
        previous
          ? ((next.plugins ?? []).find((p) => p.id === previous.id) ?? null)
          : null,
      )
    } catch (error) {
      if (request === sequence.current) setFailure(message(error))
    } finally {
      if (request === sequence.current) setLoading(false)
    }
  }
  useEffect(() => {
    if (active && !catalog && !loading && !failure) void refresh()
  }, [active])
  useEffect(
    () => () => {
      sequence.current++
    },
    [],
  )
  useEffect(() => {
    if (!selected && focusSearch.current) {
      focusSearch.current = false
      search.current?.focus()
    }
  }, [selected])
  useImperativeHandle(ref, () => ({
    command(name) {
      if (name === 'rescan') void refresh()
      if (name === 'back') setSelected(null)
      if (name === 'find') {
        window.document.documentElement.dataset.inputMode = 'keyboard'
        focusSearch.current = true
        setSelected(null)
        if (!selected) {
          focusSearch.current = false
          search.current?.focus()
        }
      }
    },
  }))
  const installed = catalog?.plugins ?? []
  const searchText = query.trim().toLocaleLowerCase()
  const items = installed.filter(
    (plugin) =>
      (!agent || plugin.agent === agent) &&
      `${plugin.name} ${plugin.description} ${plugin.version}`
        .toLocaleLowerCase()
        .includes(searchText),
  )
  const agents = ['Claude Code', 'Codex'].filter((name) =>
    installed.some((plugin) => plugin.agent === name),
  )
  return (
    <div className="plugins-browser">
      {selected ? (
        <PluginReader
          key={selected.id}
          plugin={selected}
          fontSize={fontSize}
          explorerWidth={explorerWidth}
          onBack={() => setSelected(null)}
        />
      ) : (
        <>
          <div className="library-toolbar">
            <div className="search">
              <Search size={16} />
              <input
                ref={search}
                value={query}
                placeholder={t('Search installed plugins')}
                aria-label={t('Search installed plugins')}
                onChange={(event) => setQuery(event.target.value)}
              />
              <kbd>⌘F</kbd>
            </div>
            <IconButton
              icon={RotateCw}
              label={t('Refresh installed plugins')}
              disabled={loading}
              onClick={() => void refresh()}
            />
          </div>
          <div
            className="agent-tabs plugin-agent-tabs"
            role="group"
            aria-label={t('Filter plugins by Agent')}
          >
            {['', ...agents].map((name) => (
              <button
                key={name}
                aria-pressed={agent === name}
                className={agent === name ? 'active' : ''}
                onClick={() => setAgent(name)}
              >
                {name && (
                  <AgentIcon
                    name={name}
                    icon={installed.find((p) => p.agent === name)?.icon}
                    size={16}
                  />
                )}{' '}
                {name || t('All')}{' '}
                <small>
                  {name
                    ? installed.filter((p) => p.agent === name).length
                    : installed.length}
                </small>
              </button>
            ))}
          </div>
          <div className="plugin-issues" role="status">
            {failure && <p className="error">{failure}</p>}
            {catalog?.issues?.map((issue, index) => (
              <p className="error" key={index}>
                {t(issue)}
              </p>
            ))}
          </div>
          <div className="plugin-list" aria-busy={loading}>
            {loading && !catalog ? (
              <div className="empty-library">
                <LoaderCircle className="translation-spinner" size={18} />
                <p>{t('Reading installed plugins…')}</p>
              </div>
            ) : items.length ? (
              <ul>
                {items.map((plugin) => (
                  <li className="skill-row" key={plugin.id}>
                    <button
                      className="skill-select"
                      aria-label={t('Open plugin {name}', {
                        name: plugin.name,
                      })}
                      onClick={() => setSelected(plugin)}
                    >
                      <Package size={18} className="skill-glyph" />
                      <div className="skill-info">
                        <span className="skill-name">
                          {plugin.name}{' '}
                          <small className="plugin-version">
                            {plugin.version}
                          </small>
                          {plugin.enabled === false && (
                            <small className="plugin-version">
                              {t('Disabled')}
                            </small>
                          )}
                        </span>
                        <span className="skill-description">
                          {plugin.description}
                        </span>
                      </div>
                      <span className="plugin-platform">
                        <AgentIcon
                          name={plugin.agent}
                          icon={plugin.icon}
                          size={18}
                        />
                        {plugin.agent}
                      </span>
                    </button>
                  </li>
                ))}
              </ul>
            ) : (
              <div className="empty-library">
                <Package size={28} />
                <h2>
                  {t(
                    searchText || agent
                      ? 'No matching plugins'
                      : 'No installed plugins found',
                  )}
                </h2>
                <p>
                  {t(
                    searchText || agent
                      ? 'Try another search or Agent.'
                      : 'Install a plugin in Claude Code or Codex, then refresh this list.',
                  )}
                </p>
              </div>
            )}
          </div>
          <footer className="library-footer">
            <span>
              {t('{count} installed plugins', {
                count: installed.length,
              })}
            </span>
            <span>{t('Read-only')}</span>
          </footer>
        </>
      )}
    </div>
  )
}

function PluginReader({
  plugin,
  fontSize,
  explorerWidth,
  onBack,
}: {
  plugin: Plugin
  fontSize: number
  explorerWidth: number
  onBack: () => void
}) {
  const { t, error: message } = useLocale()
  const [resources, setResources] = useState<Resource[]>([])
  const [doc, setDoc] = useState<Document | null>(null)
  const [path, setPath] = useState('')
  const [raw, setRaw] = useState(false)
  const [loading, setLoading] = useState(true)
  const [failure, setFailure] = useState('')
  const sequence = useRef(0)
  const content = useRef<HTMLDivElement>(null)
  async function read(relative: string) {
    const request = ++sequence.current
    setLoading(true)
    setFailure('')
    setPath(relative)
    setDoc(null)
    setRaw(false)
    try {
      const next = await api.ReadPlugin(plugin.id, relative)
      if (request === sequence.current) {
        setDoc(next)
        content.current?.scrollTo(0, 0)
      }
    } catch (error) {
      if (request === sequence.current) setFailure(message(error))
    } finally {
      if (request === sequence.current) setLoading(false)
    }
  }
  useEffect(() => {
    let canceled = false
    void api
      .PluginResources(plugin.id)
      .then((files) => {
        if (canceled) return
        const entries = files ?? []
        setResources(entries)
        const first =
          entries.find((file) => /^readme\.(md|markdown)$/i.test(file.path)) ??
          entries.find((file) =>
            /^(\.codex-plugin|\.claude-plugin)\/plugin\.json$/.test(file.path),
          ) ??
          entries.find((file) => file.kind === 'markdown') ??
          entries.find((file) => file.kind === 'text' || file.kind === 'image')
        if (first) void read(first.path)
        else setLoading(false)
      })
      .catch((error) => {
        if (!canceled) {
          setFailure(message(error))
          setLoading(false)
        }
      })
    return () => {
      canceled = true
      sequence.current++
    }
  }, [plugin.id])
  function openLink(href: string) {
    if (href.startsWith('#')) return
    if (/^[a-z][a-z\d+.-]*:/i.test(href)) {
      void api.OpenLink(href).catch((error) => setFailure(message(error)))
      return
    }
    try {
      const parts = path.split('/').slice(0, -1)
      for (const part of decodeURIComponent(href.split(/[?#]/)[0]).split('/')) {
        if (part === '.' || !part) continue
        if (part === '..') {
          if (!parts.length)
            throw new Error(t('This link is outside the plugin.'))
          parts.pop()
        } else parts.push(part)
      }
      void read(parts.join('/'))
    } catch (error) {
      setFailure(message(error))
    }
  }
  return (
    <aside
      className="reader plugin-reader"
      aria-label={t('Plugin reader')}
      style={{ '--reader-font-size': `${fontSize}px` } as CSSProperties}
    >
      <nav className="reader-navigation" aria-label={t('Plugin navigation')}>
        <button className="reader-back" onClick={onBack}>
          <ArrowLeft size={15} />
          {t('Back to Plugins')}
        </button>
      </nav>
      <div className="reader-body">
        <nav
          className="file-explorer"
          aria-label={t('Plugin files')}
          style={{
            width: `clamp(180px, 38vw, ${Math.max(180, Math.min(400, explorerWidth))}px)`,
          }}
        >
          <div className="plugin-heading">
            <AgentIcon name={plugin.agent} icon={plugin.icon} size={20} />
            <div>
              <strong>{plugin.name}</strong>
              <small>
                {plugin.version} · {plugin.agent}
              </small>
            </div>
          </div>
          <FileTree
            resources={resources}
            path={path}
            editable={false}
            label={t('Plugin files')}
            onOpen={(relative) => void read(relative)}
          />
        </nav>
        <section
          className="reader-document"
          aria-label={path || t('Plugin document')}
        >
          <header className="reader-head">
            <span className="filename" title={path}>
              {path}
            </span>
            {doc?.kind === 'markdown' && (
              <IconButton
                icon={raw ? Eye : FileCode}
                label={raw ? t('Read Markdown') : t('View raw file')}
                active={raw}
                onClick={() => setRaw((value) => !value)}
              />
            )}
          </header>
          {failure && (
            <div className="translation-error" role="alert">
              <span>{failure}</span>
            </div>
          )}
          <div className="reader-content" tabIndex={0} ref={content}>
            {loading ? (
              <p role="status">{t('Reading plugin file…')}</p>
            ) : doc ? (
              doc.kind === 'markdown' && !raw ? (
                <Markdown text={doc.body} onLink={openLink} />
              ) : doc.kind === 'image' ? (
                <img className="resource-image" src={doc.dataUrl} alt={path} />
              ) : doc.kind === 'binary' ? (
                <p>{t('This file cannot be previewed.')}</p>
              ) : (
                <pre className="raw-file">{doc.text}</pre>
              )
            ) : (
              !failure && <p>{t('No readable files in this plugin.')}</p>
            )}
          </div>
          <footer className="reader-footer">
            <span>
              <Lock size={11} />
              {t('Read-only')}
            </span>
          </footer>
        </section>
      </div>
    </aside>
  )
}
