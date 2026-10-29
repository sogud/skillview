import { useLocale } from './locales'
import { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import { Events } from '@wailsio/runtime'
import {
  BookOpen,
  ArrowLeft,
  Settings as SettingsIcon,
  Search,
  Plus,
  RotateCw,
  X,
  Download,
  ArchiveRestore,
  MessageSquare,
  Ellipsis,
} from 'lucide-react'
import * as api from './api'
import type {
  Catalog,
  Document,
  Resource,
  Skill,
  UpdateInfo,
  ScanProgress,
} from './api'
import { Plugins, type PluginsHandle } from './Plugins'
import { ArchiveSkillDialog, ArchivedSkillsDialog } from './ArchiveDialogs'
import { UpdateDialog } from './UpdateDialog'
import { FeedbackDialog } from './FeedbackDialog'
import { configureAutomaticErrors } from './automatic-errors'
import { IconButton, Modal } from './components'
import {
  CreateDialog,
  Settings,
  ResourceDialog,
  TrashDialog,
  type SettingsTab,
} from './Dialogs'
import { Reader, type Draft } from './Reader'
import { AgentTabs, SkillsList } from './SkillsList'
import { resolveLocalLink } from './local-link'
import { groupSkills, selectCopy, type SkillGroup } from './skill-groups'

export default function App() {
  const { t, error: message, language, setLanguage, direction } = useLocale()
  const [view, setView] = useState<'skills' | 'plugins'>('skills')
  const plugins = useRef<PluginsHandle>(null)
  const [toolbarSearch, setToolbarSearch] = useState<HTMLDivElement | null>(
    null,
  )
  const [toolbarActions, setToolbarActions] = useState<HTMLDivElement | null>(
    null,
  )
  const [actionsOpen, setActionsOpen] = useState(false)
  const actions = useRef<HTMLDivElement>(null)
  const actionsTrigger = useRef<HTMLButtonElement>(null)
  const [verified, setVerified] = useState(false)
  const [catalog, setCatalog] = useState<Catalog | null>(null)
  const [query, setQuery] = useState('')
  const [agentFilter, setAgentFilter] = useState('')
  const [translationVersion, setTranslationVersion] = useState(0)
  const [translationDisplayVersion, setTranslationDisplayVersion] = useState(0)
  const [selected, setSelected] = useState<Skill | null>(null)
  const [document, setDocument] = useState<Document | null>(null)
  const [resources, setResources] = useState<Resource[]>([])
  const [draft, setDraft] = useState<Draft | null>(null)
  const [raw, setRaw] = useState(false)
  const [loading, setLoading] = useState(false)
  const [scanning, setScanning] = useState(false)
  const [scanProgress, setScanProgress] = useState<ScanProgress | null>(null)
  const [saving, setSaving] = useState(false)
  const [readError, setReadError] = useState('')
  const [notice, setNotice] = useState<{ text: string; error: boolean } | null>(
    null,
  )
  const [theme, setTheme] = useState('system')
  const [accentColor, setAccentColor] = useState('')
  const [automaticErrors, setAutomaticErrors] = useState(false)
  const [readerLayout, setReaderLayout] = useState({
    fontSize: 14,
    width: 244,
  })
  const [feedbackOpen, setFeedbackOpen] = useState<
    'general' | 'translation' | null
  >(null)
  const [settings, setSettings] = useState<SettingsTab | null>(null)
  const [updateStatus, setUpdateStatus] = useState<UpdateInfo | null>(null)
  const [updatesOpen, setUpdatesOpen] = useState(false)
  const [diagnosticsOpen, setDiagnosticsOpen] = useState(false)
  const [resourceAction, setResourceAction] = useState<{
    kind: 'new' | 'rename'
    resource?: Resource
  } | null>(null)
  const [trashing, setTrashing] = useState<{
    path: string
    link?: { skillId: string; sourceId: string; path: string }
  } | null>(null)
  const [lastTrash, setLastTrash] = useState<string | null>(null)
  const [archiving, setArchiving] = useState(false)
  const [archivesOpen, setArchivesOpen] = useState(false)
  const [creating, setCreating] = useState(false)
  const [pendingAction, setPendingAction] = useState<(() => void) | null>(null)
  const [conflict, setConflict] = useState<Document | null>(null)
  const [conflictTab, setConflictTab] = useState('disk')
  const search = useRef<HTMLInputElement>(null)
  const readSequence = useRef(0)
  const mounted = useRef(true)
  const scanPromise = useRef<ReturnType<typeof api.Scan> | null>(null)
  const scanCanceled = useRef(false)
  const currentState = useRef({ draft, saving, selected, document })
  currentState.current = { draft, saving, selected, document }
  const dirty = !!draft && draft.text !== draft.base.text

  useEffect(() => {
    if (!actionsOpen) return
    function closeOutside(event: PointerEvent) {
      if (
        event.target instanceof Node &&
        !actions.current?.contains(event.target)
      )
        setActionsOpen(false)
    }
    window.document.addEventListener('pointerdown', closeOutside)
    return () =>
      window.document.removeEventListener('pointerdown', closeOutside)
  }, [actionsOpen])
  useEffect(() => setActionsOpen(false), [view])
  useEffect(() => {
    if (!actionsOpen) return
    const frame = requestAnimationFrame(() =>
      actions.current
        ?.querySelector<HTMLButtonElement>('[role="menuitem"]:not(:disabled)')
        ?.focus(),
    )
    return () => cancelAnimationFrame(frame)
  }, [actionsOpen])

  function closeToolbarActions() {
    setActionsOpen(false)
    // Focus the trigger before a dialog mounts so it can restore a visible control.
    if (actionsOpen) actionsTrigger.current?.focus()
  }

  useEffect(() => {
    return Events.On('skillview:scan', (event) =>
      setScanProgress(event.data as ScanProgress),
    )
  }, [])

  useEffect(() => {
    const unsubscribe = Events.On('skillview:update', (event) =>
      setUpdateStatus(event.data as UpdateInfo),
    )
    void api
      .UpdateStatus()
      .then(setUpdateStatus)
      .catch(() => {})
    return unsubscribe
  }, [])

  const notify = useCallback((text: string, error = false, undo?: string) => {
    setLastTrash(undo ?? null)
    setNotice({ text, error })
  }, [])
  const fail = useCallback(
    (error: unknown) => notify(message(error), true),
    [notify, message],
  )

  const failRef = useRef(fail)
  failRef.current = fail

  const refresh = useCallback(async () => {
    if (scanPromise.current) return scanPromise.current
    setScanning(true)
    setScanProgress(null)
    scanCanceled.current = false
    const promise = api.Scan()
    scanPromise.current = promise
    try {
      const next = await promise
      if (mounted.current) {
        setCatalog(next)
        setVerified(true)
        setSelected((previous) =>
          previous
            ? ((next.skills ?? []).find((skill) => skill.id === previous.id) ??
              previous)
            : null,
        )
      }
      return next
    } catch (failure) {
      if (!scanCanceled.current) throw failure
    } finally {
      scanPromise.current = null
      if (mounted.current) setScanning(false)
    }
  }, [])

  useEffect(() => {
    mounted.current = true
    void api
      .Preferences()
      .then((preferences) => {
        if (mounted.current) {
          setTheme(preferences.theme || 'system')
          setAccentColor(preferences.accentColor || '')
          setAutomaticErrors(Boolean(preferences.automaticErrors))
          setLanguage(preferences.language || 'system')
          setReaderLayout({
            fontSize: preferences.readerFontSize || 14,
            width: preferences.explorerWidth || 244,
          })
        }
      })
      .catch((failure) => failRef.current(failure))
    void api
      .CachedCatalog()
      .then((cached) => {
        if (mounted.current) {
          setCatalog((previous) => previous ?? cached)
          setVerified(!cached.needsScan)
        }
      })
      .catch((failure) => failRef.current(failure))
    void api.Ready().catch((failure) => failRef.current(failure))
    return () => {
      mounted.current = false
      readSequence.current++
    }
  }, [refresh, setLanguage])

  useEffect(() => {
    configureAutomaticErrors(automaticErrors, api.FeedbackConfiguration,
      (report, envelope) => api.SendAutomaticError(report, envelope))
  }, [automaticErrors])

  useEffect(() => {
    window.document.documentElement.dataset.theme = theme
  }, [theme])
  useEffect(() => {
    const style = window.document.documentElement.style
    if (!accentColor) {
      style.removeProperty('--accent')
      style.removeProperty('--accent-text')
      return
    }
    style.setProperty('--accent', accentColor)
    const red = parseInt(accentColor.slice(1, 3), 16)
    const green = parseInt(accentColor.slice(3, 5), 16)
    const blue = parseInt(accentColor.slice(5, 7), 16)
    style.setProperty(
      '--accent-text',
      red * 0.299 + green * 0.587 + blue * 0.114 > 160 ? '#17191e' : '#ffffff',
    )
  }, [accentColor])
  useEffect(() => {
    if (!notice || notice.error) return
    const timer = window.setTimeout(
      () => setNotice(null),
      lastTrash ? 10000 : 2800,
    )
    return () => window.clearTimeout(timer)
  }, [notice, lastTrash])

  useEffect(() => {
    const onFocus = async () => {
      if (!currentState.current.saving) {
        const state = currentState.current
        try {
          const cached = await api.CachedCatalog()
          if (mounted.current && !currentState.current.saving)
            setCatalog(cached)
        } catch {
          /* The current document and draft remain available. */
        }
        if (!state.selected || !state.document || state.draft) return
        const sequence = readSequence.current
        try {
          const [next, list] = await Promise.all([
            api.Read(state.selected.id, state.document.path),
            api.Resources(state.selected.id),
          ])
          if (
            mounted.current &&
            sequence === readSequence.current &&
            !currentState.current.draft
          ) {
            setDocument(next)
            setResources(list ?? [])
            setReadError('')
          }
        } catch (failure) {
          if (
            mounted.current &&
            sequence === readSequence.current &&
            !currentState.current.draft
          )
            setReadError(message(failure))
        }
      }
    }
    window.addEventListener('focus', onFocus)
    const unsubscribe = Events.On('skillview:activated', onFocus)
    return () => {
      window.removeEventListener('focus', onFocus)
      unsubscribe()
    }
  }, [message])

  function request(action: () => void) {
    if (saving) return
    if (dirty) {
      setPendingAction(() => action)
      return
    }
    action()
  }

  function checkUpdates() {
    void api.CheckUpdates().then(setUpdateStatus).catch(fail)
  }

  function openUpdates() {
    setSettings(null)
    setUpdatesOpen(true)
    if (!updateStatus?.latest) checkUpdates()
  }

  async function load(skill: Skill, path = 'SKILL.md', edit = false) {
    const sequence = ++readSequence.current
    setSelected(skill)
    setDocument(null)
    setDraft(null)
    setRaw(false)
    setLoading(true)
    setReadError('')
    const results = await Promise.allSettled([
      api.Read(skill.id, path),
      api.Resources(skill.id),
    ])
    if (!mounted.current || sequence !== readSequence.current) return
    const read = results[0]
    const resourceList = results[1]
    if (read.status === 'fulfilled') {
      setDocument(read.value)
      if (edit && read.value.editable)
        setDraft({ base: read.value, text: read.value.text })
    } else setReadError(message(read.reason))
    setResources(
      resourceList.status === 'fulfilled' ? (resourceList.value ?? []) : [],
    )
    if (resourceList.status === 'rejected') fail(resourceList.reason)
    setLoading(false)
  }

  function openResource(path: string) {
    if (selected)
      request(() => {
        void load(selected, path)
      })
  }
  function closeReader() {
    request(leaveReader)
  }
  function leaveReader() {
    readSequence.current++
    setSelected(null)
    setDocument(null)
    setDraft(null)
    setConflict(null)
  }

  async function save() {
    if (!draft || !dirty || saving || !selected) return
    setSaving(true)
    try {
      const result = await api.Save(
        selected.id,
        draft.base.path,
        draft.base.revision,
        draft.text,
      )
      if (result.status === 'conflict') {
        setConflict(result.document)
        setConflictTab('disk')
      } else {
        setDocument(result.document)
        setDraft({ base: result.document, text: result.document.text })
        notify(t('Saved'))
        await refresh()
      }
    } catch (error) {
      fail(error)
    } finally {
      setSaving(false)
    }
  }

  function openLink(url: string) {
    if (url.startsWith('#')) return
    if (/^[a-z][a-z\d+.-]*:/i.test(url)) {
      void api.OpenLink(url).catch(fail)
      return
    }
    try {
      const path = resolveLocalLink(document?.path ?? 'SKILL.md', url)
      if (path === null) {
        notify(
          t(
            'This link is outside the current Skill. Open the folder in Finder.',
          ),
          true,
        )
        return
      }
      openResource(path)
    } catch (error) {
      fail(error)
    }
  }

  async function addSource(writable: boolean) {
    if (await api.ChooseSource(writable)) await refresh()
  }
  async function create(sourceId: string, name: string, description: string) {
    const id = await api.Create(sourceId, name, description)
    const next = await refresh()
    const skill = next?.skills?.find((item) => item.id === id)
    setQuery('')
    setAgentFilter('')
    setCreating(false)
    if (skill) await load(skill, 'SKILL.md', true)
    notify(t('Skill created'))
  }

  const commands = useRef<(command: string) => void>(() => {})
  commands.current = (command) => {
    if (view === 'plugins' && ['find', 'rescan'].includes(command)) {
      plugins.current?.command(command)
      return
    }
    if (command === 'updates') openUpdates()
    if (command === 'find') {
      window.document.documentElement.dataset.inputMode = 'keyboard'
      request(() => {
        leaveReader()
        requestAnimationFrame(() => search.current?.focus())
      })
    }
    if (command === 'settings') setSettings('general')
    if (command === 'new')
      request(() => {
        setView('skills')
        setCreating(true)
      })
    if (command === 'save') void save()
    if (command === 'rescan') void refresh().catch(fail)
    if (command === 'quit')
      request(() => {
        void api.Quit().catch(fail)
      })
  }
  useEffect(
    () =>
      Events.On('skillview:command', (event) =>
        commands.current(String(event.data)),
      ),
    [],
  )

  const groups = useMemo(() => groupSkills(catalog?.skills ?? []), [catalog])
  const skills = useMemo(() => {
    const needle = query.trim().toLocaleLowerCase()
    return groups.filter((group) => {
      if (agentFilter && !group.tools.includes(agentFilter)) return false
      return (
        !needle ||
        `${group.name} ${group.description}`
          .toLocaleLowerCase()
          .includes(needle)
      )
    })
  }, [groups, query, agentFilter])
  const selectedGroup = selected
    ? groups.find((group) =>
        group.members.some((copy) => copy.id === selected.id),
      )
    : undefined

  function openGroup(group: SkillGroup) {
    request(() => {
      void load(selectCopy(group, agentFilter, selected?.id))
    })
  }

  function changeAgent(agent: string) {
    request(() => {
      setAgentFilter(agent)
      if (selectedGroup && (!agent || selectedGroup.tools.includes(agent))) {
        void load(selectCopy(selectedGroup, agent, selected?.id))
      } else if (selected) {
        readSequence.current++
        setSelected(null)
        setDocument(null)
        setDraft(null)
      }
    })
  }

  return (
    <div
      className="app"
      onKeyDown={(event) => {
        if (
          event.key === 'Escape' &&
          !settings &&
          !feedbackOpen &&
          !updatesOpen &&
          !creating &&
          !conflict &&
          !pendingAction &&
          !resourceAction &&
          !trashing &&
          !archiving &&
          !archivesOpen
        )
          view === 'plugins' ? plugins.current?.command('back') : closeReader()
      }}
    >
      <div className="titlebar">
        <div className="toolbar-search" dir={direction} ref={setToolbarSearch}>
          {view === 'skills' && selected && (
            <button
              className="reader-back"
              onClick={closeReader}
              disabled={saving}
              title={t('Back to Skills')}
            >
              <ArrowLeft size={15} />
              <span>{t('Back to Skills')}</span>
            </button>
          )}
          {view === 'skills' && !selected && (
            <div className="search">
              <Search size={14} />
              <input
                ref={search}
                aria-label={t('Search Skills')}
                placeholder={t('Search Skills')}
                value={query}
                onChange={(event) => setQuery(event.target.value)}
              />
              <kbd>⌘F</kbd>
            </div>
          )}
        </div>
        <div
          className="view-switch"
          dir={direction}
          role="tablist"
          aria-label={t('Library view')}
        >
          {(['skills', 'plugins'] as const).map((mode) => (
            <button
              key={mode}
              id={`${mode}-tab`}
              title={t(mode === 'skills' ? 'Skills' : 'Plugins')}
              role="tab"
              aria-selected={view === mode}
              aria-controls={`${mode}-view`}
              tabIndex={view === mode ? 0 : -1}
              onClick={() => request(() => setView(mode))}
              onKeyDown={(event) => {
                if (['ArrowLeft', 'ArrowRight'].includes(event.key)) {
                  event.preventDefault()
                  const next = mode === 'skills' ? 'plugins' : 'skills'
                  request(() => {
                    setView(next)
                    requestAnimationFrame(() =>
                      window.document.getElementById(`${next}-tab`)?.focus(),
                    )
                  })
                }
              }}
            >
              {t(mode === 'skills' ? 'Skills' : 'Plugins')}
            </button>
          ))}
        </div>
        <div
          className={`window-actions${actionsOpen ? ' open' : ''}`}
          dir={direction}
          ref={actions}
          onKeyDown={(event) => {
            if (
              actionsOpen &&
              ['ArrowDown', 'ArrowUp', 'Home', 'End'].includes(event.key)
            ) {
              event.preventDefault()
              const items = Array.from(
                event.currentTarget.querySelectorAll<HTMLButtonElement>(
                  '[role="menuitem"]:not(:disabled)',
                ),
              )
              const current = items.indexOf(
                window.document.activeElement as HTMLButtonElement,
              )
              const next =
                event.key === 'Home'
                  ? 0
                  : event.key === 'End'
                    ? items.length - 1
                    : (current +
                        (event.key === 'ArrowDown' ? 1 : -1) +
                        items.length) %
                      items.length
              items[next]?.focus()
            }
            if (event.key === 'Escape' && actionsOpen) {
              event.preventDefault()
              event.stopPropagation()
              setActionsOpen(false)
              actionsTrigger.current?.focus()
            }
          }}
          onBlur={(event) => {
            // WebKit can blur to the document before a menu button's click.
            // Outside pointer presses already close the menu above.
            if (
              event.relatedTarget &&
              !event.currentTarget.contains(event.relatedTarget)
            )
              setActionsOpen(false)
          }}
        >
          {view === 'skills' && !selected && (
            <IconButton
              icon={Plus}
              label={t('New Skill')}
              onClick={() => request(() => setCreating(true))}
              disabled={scanning || saving}
            />
          )}
          <button
            type="button"
            className={`icon-button toolbar-overflow${updateStatus?.latest ? ' has-update' : ''}`}
            ref={actionsTrigger}
            aria-label={
              updateStatus?.latest
                ? `${t('More actions')} · ${t('Update available')}`
                : t('More actions')
            }
            title={
              updateStatus?.latest
                ? `${t('More actions')} · ${t('Update available')}`
                : t('More actions')
            }
            aria-haspopup="menu"
            aria-expanded={actionsOpen}
            aria-controls="toolbar-actions"
            onKeyDown={(event) => {
              if (event.key === 'ArrowDown' || event.key === 'ArrowUp') {
                event.preventDefault()
                setActionsOpen(true)
              }
            }}
            onClick={() => setActionsOpen((open) => !open)}
          >
            <Ellipsis size={16} />
          </button>
          <div
            className="toolbar-action-items"
            id="toolbar-actions"
            role="menu"
            aria-label={t('More actions')}
            onClick={(event) => {
              if ((event.target as HTMLElement).closest('button'))
                closeToolbarActions()
            }}
          >
            <div className="toolbar-view-actions" ref={setToolbarActions}>
              {view === 'skills' && !selected && (
                <>
                  <button
                    type="button"
                    role="menuitem"
                    onClick={() => {
                      void refresh().catch(fail)
                    }}
                    disabled={scanning}
                  >
                    <RotateCw size={16} />
                    <span>{t('Rescan folders')}</span>
                  </button>
                  <button
                    type="button"
                    role="menuitem"
                    onClick={() => request(() => setArchivesOpen(true))}
                    disabled={scanning || saving}
                  >
                    <ArchiveRestore size={16} />
                    <span>{t('Archived Skills')}</span>
                  </button>
                </>
              )}
            </div>
            {!!updateStatus?.latest && (
              <button type="button" role="menuitem" onClick={openUpdates}>
                <Download size={16} />
                <span>
                  {t(
                    updateStatus.state === 'ready'
                      ? 'Restart to update'
                      : 'Update available',
                  )}
                </span>
              </button>
            )}
            <button
              type="button"
              role="menuitem"
              onClick={() => setFeedbackOpen('general')}
            >
              <MessageSquare size={16} />
              <span>{t('Send feedback')}</span>
            </button>
            <button
              type="button"
              role="menuitem"
              onClick={() => setSettings('general')}
            >
              <SettingsIcon size={16} />
              <span>{t('Settings')}</span>
            </button>
          </div>
        </div>
      </div>
      <div className="shell">
        <main className={`workspace${selected ? ' expanded' : ''}`}>
          <div
            className="skills-view"
            id="skills-view"
            role="tabpanel"
            aria-labelledby="skills-tab"
            hidden={view !== 'skills'}
          >
            <section
              className="library"
              hidden={!!selected}
              aria-label={t('Skills library')}
            >
              <AgentTabs
                agents={catalog?.agents ?? []}
                groups={groups}
                value={agentFilter}
                onChange={changeAgent}
              />
              {scanning && (
                <div className="scan-status" role="status" aria-live="polite">
                  <div>
                    <strong>{t('Reading Skills…')}</strong>
                    <span>
                      {t('{done} / {total} folders · {count} Skills found', {
                        done: scanProgress?.sourcesDone ?? 0,
                        total: scanProgress?.sourcesTotal ?? 0,
                        count: scanProgress?.skills ?? 0,
                      })}
                    </span>
                    <button
                      onClick={() => {
                        scanCanceled.current = true
                        scanPromise.current?.cancel()
                      }}
                    >
                      {t('Cancel scan')}
                    </button>
                  </div>
                  <progress
                    aria-label={t('Scan progress')}
                    value={
                      scanProgress?.phase === 'reading' &&
                      scanProgress.sourcesTotal > 0
                        ? scanProgress.sourcesDone
                        : undefined
                    }
                    max={scanProgress?.sourcesTotal || 1}
                  />
                </div>
              )}
              {!scanning && catalog?.needsScan && !!catalog.skills?.length && (
                <p className="scan-hint">
                  {t('Scan once to reopen your saved Skills.')}
                </p>
              )}
              <SkillsList
                groups={skills}
                agents={catalog?.agents ?? []}
                selectedId={selected?.id}
                verified={verified}
                scanning={scanning}
                onOpen={openGroup}
                empty={
                  <div className="empty-library">
                    <BookOpen size={30} strokeWidth={1.2} />
                    <h2>
                      {scanning
                        ? t('Reading your Skills folders…')
                        : query
                          ? t('No Skills match your search')
                          : verified
                            ? t('No active Skills in this view')
                            : t('Scan your Skills folders')}
                    </h2>
                    <p>
                      {query
                        ? t('Try a Skill name or a word from its description.')
                        : verified
                          ? t(
                              'Scan folders or restore a Skill from the archive.',
                            )
                          : t(
                              'Scanning only starts when you click Scan folders.',
                            )}
                    </p>
                    {!query && !scanning && (
                      <button
                        className="primary"
                        onClick={() => {
                          void refresh().catch(fail)
                        }}
                      >
                        {t('Scan folders')}
                      </button>
                    )}
                    {!query && !scanning && verified && (
                      <button
                        onClick={() => request(() => setArchivesOpen(true))}
                      >
                        {t('Archived Skills')}
                      </button>
                    )}
                  </div>
                }
              />
              <footer className="library-footer">
                <span>
                  {scanning
                    ? t('Scanning…')
                    : t('{count} Skills', { count: skills.length })}
                </span>
                {!!catalog?.issues?.length && (
                  <button onClick={() => setSettings('folders')}>
                    {t('Some paths need attention')}
                  </button>
                )}
              </footer>
            </section>
            {selected && (
              <Reader
                skill={selected}
                copies={selectedGroup?.members ?? [selected]}
                versions={selectedGroup?.versions.length ?? 1}
                onCopy={(copy) =>
                  request(() => {
                    void load(copy)
                  })
                }
                translationVersion={translationVersion}
                translationDisplayVersion={translationDisplayVersion}
                fontSize={readerLayout.fontSize}
                explorerWidth={readerLayout.width}
                onLayoutChange={(fontSize, width) => {
                  setReaderLayout({ fontSize, width })
                  void api.SetReaderLayout(fontSize, width).catch(fail)
                }}
                onConfigureTranslation={() => setSettings('translation')}
                onReportTranslation={() => setFeedbackOpen('translation')}
                onNewResource={() =>
                  request(() => setResourceAction({ kind: 'new' }))
                }
                onRenameResource={(resource) =>
                  request(() => setResourceAction({ kind: 'rename', resource }))
                }
                onImportResource={() =>
                  request(() => {
                    void api
                      .ChooseResource(selected.id)
                      .then(async (path) => {
                        if (path) {
                          await refresh()
                          await load(selected, path)
                        }
                      })
                      .catch(fail)
                  })
                }
                onTrashResource={(resource) =>
                  request(() => setTrashing({ path: resource.path }))
                }
                onTrashSkill={() => request(() => setTrashing({ path: '' }))}
                onArchiveSkill={() => request(() => setArchiving(true))}
                onRemoveLink={(copy, occurrence) =>
                  request(() =>
                    setTrashing({
                      path: '',
                      link: {
                        skillId: copy.id,
                        sourceId: occurrence.sourceId,
                        path: occurrence.path,
                      },
                    }),
                  )
                }
                document={document}
                resources={resources}
                draft={draft}
                raw={raw}
                loading={loading}
                error={readError}
                saving={saving}
                onRaw={() => setRaw((value) => !value)}
                onEdit={() => {
                  if (document?.editable)
                    setDraft({ base: document, text: document.text })
                }}
                onChange={(text) =>
                  setDraft((previous) =>
                    previous ? { ...previous, text } : null,
                  )
                }
                onSave={() => {
                  void save()
                }}
                onCancel={() => request(() => setDraft(null))}
                onResource={openResource}
                onLink={openLink}
                onReveal={() => {
                  void api.Reveal(selected.id, document?.path ?? '').catch(fail)
                }}
              />
            )}
          </div>
          <section
            className="plugins-view"
            id="plugins-view"
            role="tabpanel"
            aria-labelledby="plugins-tab"
            hidden={view !== 'plugins'}
          >
            <Plugins
              ref={plugins}
              active={view === 'plugins'}
              toolbarSearch={toolbarSearch}
              toolbarActions={toolbarActions}
              onToolbarAction={closeToolbarActions}
              fontSize={readerLayout.fontSize}
              explorerWidth={readerLayout.width}
            />
          </section>
        </main>
      </div>
      {notice && (
        <div
          className={`notice${notice.error ? ' error' : ''}`}
          role={notice.error ? 'alert' : 'status'}
        >
          <span>{notice.text}</span>
          {lastTrash && (
            <button
              onClick={() => {
                void api
                  .UndoTrash(lastTrash)
                  .then(async () => {
                    setLastTrash(null)
                    notify(t('Restored'))
                    await refresh()
                  })
                  .catch(fail)
              }}
            >
              {t('Undo')}
            </button>
          )}
          <IconButton
            icon={X}
            label={t('Dismiss message')}
            onClick={() => setNotice(null)}
          />
        </div>
      )}
      {feedbackOpen && (
        <FeedbackDialog
          initialIncludeAIError={feedbackOpen === 'translation'}
          onClose={() => setFeedbackOpen(null)}
        />
      )}
      {settings && (
        <Settings
          version={updateStatus?.version ?? '…'}
          onUpdates={openUpdates}
          openDiagnostics={diagnosticsOpen}
          tab={settings}
          theme={theme}
          accentColor={accentColor}
          automaticErrors={automaticErrors}
          onAutomaticErrors={async (enabled) => {
            await api.SetAutomaticErrors(enabled)
            setAutomaticErrors(enabled)
          }}
          onAccentColor={(value) => {
            void api
              .SetAccentColor(value)
              .then(() => setAccentColor(value))
              .catch(fail)
          }}
          language={language}
          onLanguage={(value) => {
            void api
              .SetLanguage(value)
              .then(() => setLanguage(value))
              .catch(fail)
          }}
          onTranslationSaved={(invalidate = true) => {
            setTranslationDisplayVersion((value) => value + 1)
            if (invalidate) setTranslationVersion((value) => value + 1)
          }}
          catalog={catalog}
          onTab={setSettings}
          onTheme={(value) => {
            void api
              .SetTheme(value)
              .then(() => setTheme(value))
              .catch(fail)
          }}
          onAdd={addSource}
          onRemove={async (id) => {
            await api.RemoveSource(id)
            await refresh()
          }}
          onRescan={() => {
            void refresh().catch(fail)
          }}
          onClose={() => {
            setSettings(null)
            setDiagnosticsOpen(false)
          }}
        />
      )}
      {updatesOpen && (
        <UpdateDialog
          status={updateStatus}
          onDiagnostics={() => {
            setDiagnosticsOpen(true)
            setUpdatesOpen(false)
            setSettings('general')
          }}
          onCheck={checkUpdates}
          onDownload={() => {
            void api.DownloadUpdate().then(setUpdateStatus).catch(fail)
          }}
          onRestart={() => {
            setUpdatesOpen(false)
            request(() => {
              void api.RestartUpdate().catch(fail)
            })
          }}
          onLink={openLink}
          onClose={() => setUpdatesOpen(false)}
        />
      )}
      {resourceAction && selected && (
        <ResourceDialog
          kind={resourceAction.kind}
          resource={resourceAction.resource}
          skillId={selected.id}
          onClose={() => setResourceAction(null)}
          onSubmit={async (path, text) => {
            if (resourceAction.kind === 'rename' && resourceAction.resource)
              await api.RenameResource(
                selected.id,
                resourceAction.resource.path,
                path,
                resourceAction.resource.revision,
              )
            else await api.CreateResource(selected.id, path, text)
            setResourceAction(null)
            await refresh()
            await load(selected, path)
          }}
        />
      )}
      {archiving && selected && (
        <ArchiveSkillDialog
          skillId={selected.id}
          onClose={() => setArchiving(false)}
          onArchive={async (token) => {
            const receipt = await api.ArchiveSkill(selected.id, token)
            setArchiving(false)
            readSequence.current++
            setSelected(null)
            setDocument(null)
            setDraft(null)
            await refresh()
            notify(
              receipt.warning ? message(receipt.warning) : t('Skill archived.'),
              !!receipt.warning,
            )
          }}
        />
      )}
      {archivesOpen && (
        <ArchivedSkillsDialog
          onClose={() => setArchivesOpen(false)}
          onRestored={async () => {
            await refresh()
          }}
        />
      )}
      {trashing && selected && (
        <TrashDialog
          skillId={selected.id}
          path={trashing.path}
          link={trashing.link}
          onClose={() => setTrashing(null)}
          onTrash={async (token) => {
            const receipt = trashing.link
              ? await api.TrashLink(
                  trashing.link.skillId,
                  trashing.link.sourceId,
                  trashing.link.path,
                  token,
                )
              : await api.Trash(selected.id, trashing.path, token)
            setLastTrash(receipt.id)
            setTrashing(null)
            if (!trashing.path) {
              readSequence.current++
              setSelected(null)
              setDocument(null)
              setDraft(null)
            } else await load(selected)
            await refresh()
            notify(
              receipt.warning ? message(receipt.warning) : t('Moved to Trash'),
              !!receipt.warning,
              receipt.id,
            )
          }}
        />
      )}
      {creating && (
        <CreateDialog
          sources={catalog?.sources ?? []}
          onCreate={create}
          onAddSource={async () => {
            await addSource(true)
          }}
          onClose={() => setCreating(false)}
        />
      )}
      {pendingAction && (
        <Modal
          title={t('Discard unsaved changes?')}
          onClose={() => setPendingAction(null)}
        >
          <div className="modal-body">
            <p>
              {t(
                'Your draft has not been saved to {path}. Keep editing to save it first.',
                { path: draft?.base.path ?? '' },
              )}
            </p>
            <div className="modal-actions">
              <button
                className="primary"
                onClick={() => setPendingAction(null)}
              >
                {t('Keep editing')}
              </button>
              <button
                className="danger"
                onClick={() => {
                  const action = pendingAction
                  setPendingAction(null)
                  setDraft(null)
                  setConflict(null)
                  action()
                }}
              >
                {t('Discard draft')}
              </button>
            </div>
          </div>
        </Modal>
      )}
      {conflict && draft && (
        <Modal
          title={t('File changed outside SkillView')}
          onClose={() => setConflict(null)}
        >
          <div className="modal-body">
            <p>
              {t(
                'The disk version was kept. Review the three versions before continuing.',
              )}
            </p>
            <div
              className="segmented"
              role="group"
              aria-label={t('Conflict versions')}
            >
              {['baseline', 'disk', 'draft'].map((version) => (
                <button
                  key={version}
                  aria-pressed={conflictTab === version}
                  onClick={() => setConflictTab(version)}
                >
                  {version === 'baseline'
                    ? t('Original')
                    : version === 'disk'
                      ? t('On disk')
                      : t('Your draft')}
                </button>
              ))}
            </div>
            <pre className="conflict-text">
              {conflictTab === 'baseline'
                ? draft.base.text
                : conflictTab === 'disk'
                  ? conflict.text
                  : draft.text}
            </pre>
            <div className="modal-actions">
              <button
                onClick={() => {
                  setDocument(conflict)
                  setDraft(null)
                  setConflict(null)
                }}
              >
                {t('Use disk version')}
              </button>
              <button
                className="primary"
                onClick={() => {
                  setDraft({ base: conflict, text: draft.text })
                  setDocument(conflict)
                  setConflict(null)
                }}
              >
                {t('Review and merge draft')}
              </button>
            </div>
            <p className="quiet-note">
              {t(
                'Continue editing to merge the disk changes into your draft, then save.',
              )}
            </p>
          </div>
        </Modal>
      )}
    </div>
  )
}
