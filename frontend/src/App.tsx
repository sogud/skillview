import { useLocale } from './locales'
import { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import { Events } from '@wailsio/runtime'
import {
  BookOpen,
  Star,
  Folder,
  Settings as SettingsIcon,
  Search,
  Plus,
  RotateCw,
  X,
} from 'lucide-react'
import * as api from './api'
import type { Catalog, Document, Resource, Skill } from './api'
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
import { groupSkills, selectCopy, type SkillGroup } from './skill-groups'

export default function App() {
  const { t, error: message, language, setLanguage } = useLocale()
  const [verified, setVerified] = useState(false)
  const [catalog, setCatalog] = useState<Catalog | null>(null)
  const [query, setQuery] = useState('')
  const [favoritesOnly, setFavoritesOnly] = useState(false)
  const [agentFilter, setAgentFilter] = useState('')
  const [translationVersion, setTranslationVersion] = useState(0)
  const [selected, setSelected] = useState<Skill | null>(null)
  const [document, setDocument] = useState<Document | null>(null)
  const [resources, setResources] = useState<Resource[]>([])
  const [draft, setDraft] = useState<Draft | null>(null)
  const [raw, setRaw] = useState(false)
  const [tab, setTab] = useState<'document' | 'resources'>('document')
  const [loading, setLoading] = useState(false)
  const [scanning, setScanning] = useState(false)
  const [saving, setSaving] = useState(false)
  const [readError, setReadError] = useState('')
  const [notice, setNotice] = useState<{ text: string; error: boolean } | null>(
    null,
  )
  const [theme, setTheme] = useState('system')
  const [settings, setSettings] = useState<SettingsTab | null>(null)
  const [resourceAction, setResourceAction] = useState<{
    kind: 'new' | 'rename'
    resource?: Resource
  } | null>(null)
  const [trashing, setTrashing] = useState<{
    path: string
    link?: { skillId: string; sourceId: string; path: string }
  } | null>(null)
  const [lastTrash, setLastTrash] = useState<string | null>(null)
  const [creating, setCreating] = useState(false)
  const [pendingAction, setPendingAction] = useState<(() => void) | null>(null)
  const [conflict, setConflict] = useState<Document | null>(null)
  const [conflictTab, setConflictTab] = useState('disk')
  const search = useRef<HTMLInputElement>(null)
  const readSequence = useRef(0)
  const mounted = useRef(true)
  const scanPromise = useRef<Promise<Catalog> | null>(null)
  const currentState = useRef({ draft, saving, selected, document })
  currentState.current = { draft, saving, selected, document }
  const dirty = !!draft && draft.text !== draft.base.text

  const notify = useCallback((text: string, error = false, undo?: string) => {
    setLastTrash(undo ?? null)
    setNotice({ text, error })
  }, [])
  const fail = useCallback(
    (error: unknown) => notify(message(error), true),
    [notify, message],
  )

  const refresh = useCallback(async (discover = false) => {
    if (scanPromise.current) return scanPromise.current
    setScanning(true)
    const promise = discover ? api.Discover() : api.Scan()
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
          setLanguage(preferences.language || 'system')
        }
      })
      .catch(fail)
    void api
      .CachedCatalog()
      .then((cached) => {
        if (mounted.current && cached.scannedAt)
          setCatalog((previous) => previous ?? cached)
      })
      .catch(() => {})
    void refresh().catch(fail)
    void api.Ready().catch(fail)
    return () => {
      mounted.current = false
      readSequence.current++
    }
  }, [fail, refresh])

  useEffect(() => {
    window.document.documentElement.dataset.theme = theme
  }, [theme])
  useEffect(() => {
    if (!notice || notice.error) return
    const timer = window.setTimeout(
      () => setNotice(null),
      lastTrash ? 10000 : 2800,
    )
    return () => window.clearTimeout(timer)
  }, [notice, lastTrash])

  useEffect(() => {
    const onFocus = () => {
      if (!currentState.current.saving)
        void refresh()
          .then(async () => {
            const state = currentState.current
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
          })
          .catch(fail)
    }
    const unsubscribe = Events.On('skillview:catalog', onFocus)
    window.addEventListener('focus', onFocus)
    return () => {
      unsubscribe()
      window.removeEventListener('focus', onFocus)
    }
  }, [fail, refresh])

  function request(action: () => void) {
    if (saving) return
    if (dirty) {
      setPendingAction(() => action)
      return
    }
    action()
  }

  async function load(skill: Skill, path = 'SKILL.md', edit = false) {
    const sequence = ++readSequence.current
    setSelected(skill)
    setTab('document')
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
    request(() => {
      readSequence.current++
      setSelected(null)
      setDocument(null)
      setDraft(null)
      setConflict(null)
    })
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
      const parts = [
        ...(document?.path.split('/').slice(0, -1) ?? []),
        ...decodeURIComponent(url.split(/[?#]/)[0]).split('/'),
      ]
      if (parts.some((part) => !part || part === '.' || part === '..')) {
        notify(
          t(
            'This link is outside the current Skill. Open the folder in Finder.',
          ),
          true,
        )
        return
      }
      openResource(parts.join('/'))
    } catch (error) {
      fail(error)
    }
  }

  async function favorite(group: SkillGroup) {
    const value = !group.favorite
    const ids = group.members.map((copy) => copy.id)
    try {
      await api.SetFavorites(ids, value)
      setCatalog((previous) =>
        previous
          ? {
              ...previous,
              skills:
                previous.skills?.map((item) =>
                  ids.includes(item.id) ? { ...item, favorite: value } : item,
                ) ?? [],
            }
          : previous,
      )
      setSelected((previous) =>
        previous && ids.includes(previous.id)
          ? { ...previous, favorite: value }
          : previous,
      )
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
    const skill = next.skills?.find((item) => item.id === id)
    setQuery('')
    setFavoritesOnly(false)
    setAgentFilter('')
    setCreating(false)
    if (skill) await load(skill, 'SKILL.md', true)
    notify(t('Skill created'))
  }

  const commands = useRef<(command: string) => void>(() => {})
  commands.current = (command) => {
    if (command === 'find') {
      window.document.documentElement.dataset.inputMode = 'keyboard'
      search.current?.focus()
    }
    if (command === 'settings') setSettings('general')
    if (command === 'new') request(() => setCreating(true))
    if (command === 'save') void save()
    if (command === 'rescan') void refresh(true).catch(fail)
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
      if (favoritesOnly && !group.favorite) return false
      if (agentFilter && !group.tools.includes(agentFilter)) return false
      return (
        !needle ||
        `${group.name} ${group.description}`
          .toLocaleLowerCase()
          .includes(needle)
      )
    })
  }, [groups, favoritesOnly, query, agentFilter])
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
          !creating &&
          !conflict &&
          !pendingAction &&
          !resourceAction &&
          !trashing
        )
          closeReader()
      }}
    >
      <div className="titlebar">
        <span>{t('Skills')}</span>
      </div>
      <div className="shell">
        <nav className="rail" aria-label={t('Navigation')}>
          <IconButton
            icon={BookOpen}
            label={t('All Skills')}
            active={!favoritesOnly}
            onClick={() => setFavoritesOnly(false)}
          />
          <IconButton
            icon={Star}
            label={t('Favorite Skills')}
            active={favoritesOnly}
            onClick={() => setFavoritesOnly(true)}
          />
          <IconButton
            icon={Folder}
            label={t('Sources')}
            active={settings === 'sources'}
            onClick={() => setSettings('sources')}
          />
          <div className="rail-spacer" />
          <IconButton
            icon={SettingsIcon}
            label={t('Settings')}
            onClick={() => setSettings('general')}
          />
        </nav>
        <main className={`workspace${selected ? ' expanded' : ''}`}>
          <section className="library" aria-label={t('Skills library')}>
            <div className="library-toolbar">
              <div className="search">
                <Search size={14} />
                <input
                  ref={search}
                  aria-label={t('Search Skills')}
                  placeholder={
                    favoritesOnly ? t('Search favorites') : t('Search Skills')
                  }
                  value={query}
                  onChange={(event) => setQuery(event.target.value)}
                />
                <kbd>⌘F</kbd>
              </div>
              <IconButton
                icon={RotateCw}
                label={t('Rescan folders')}
                onClick={() => {
                  void refresh(true).catch(fail)
                }}
                disabled={scanning}
              />
              <IconButton
                icon={Plus}
                label={t('New Skill')}
                onClick={() => request(() => setCreating(true))}
                disabled={scanning || saving}
              />
            </div>
            <AgentTabs
              agents={catalog?.agents ?? []}
              groups={groups}
              value={agentFilter}
              onChange={changeAgent}
            />
            <SkillsList
              groups={skills}
              agents={catalog?.agents ?? []}
              selectedId={selected?.id}
              verified={verified}
              scanning={scanning}
              onOpen={openGroup}
              onFavorite={(group) => {
                void favorite(group)
              }}
              empty={
                <div className="empty-library">
                  <BookOpen size={30} strokeWidth={1.2} />
                  <h2>
                    {scanning
                      ? t('Reading your Skills folders…')
                      : query
                        ? t('No Skills match your search')
                        : favoritesOnly
                          ? t('Your favorites will appear here')
                          : t('Add your Skills folders')}
                  </h2>
                  <p>
                    {query
                      ? t('Try a Skill name or a word from its description.')
                      : favoritesOnly
                        ? t('Use the star beside a Skill to save it here.')
                        : t('Choose an existing folder to start reading.')}
                  </p>
                  {!query && !favoritesOnly && !scanning && (
                    <button
                      className="primary"
                      onClick={() => setSettings('sources')}
                    >
                      {t('Add Skills folder')}
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
              <button onClick={() => setSettings('sources')}>
                {(catalog?.issues?.length ?? 0) > 0
                  ? t('Some paths need attention')
                  : t('Manage sources')}
              </button>
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
              onConfigureTranslation={() => setSettings('translation')}
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
              tab={tab}
              raw={raw}
              loading={loading}
              error={readError}
              saving={saving}
              onTab={setTab}
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
              onClose={closeReader}
            />
          )}
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
      {settings && (
        <Settings
          tab={settings}
          theme={theme}
          language={language}
          onLanguage={(value) => {
            void api
              .SetLanguage(value)
              .then(() => setLanguage(value))
              .catch(fail)
          }}
          onTranslationSaved={() => setTranslationVersion((value) => value + 1)}
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
            void refresh(true).catch(fail)
          }}
          onClose={() => setSettings(null)}
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
