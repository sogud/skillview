import { useEffect, useRef, useState, type ReactNode } from 'react'
import {
  FileText,
  Link,
  AlertCircle,
  Star,
  ChevronLeft,
  ChevronRight,
} from 'lucide-react'
import { Virtuoso, type VirtuosoHandle } from 'react-virtuoso'
import type { AgentInfo } from './api'
import { AgentIcon } from './AgentIcon'
import { IconButton } from './components'
import { useLocale } from './locales'
import { platformNames, type SkillGroup } from './skill-groups'

type Props = {
  groups: SkillGroup[]
  agents: AgentInfo[]
  selectedId?: string
  verified: boolean
  scanning: boolean
  onOpen: (group: SkillGroup) => void
  onFavorite: (group: SkillGroup) => void
  empty: ReactNode
}

export function SkillsList(props: Props) {
  const { t } = useLocale()
  const list = useRef<VirtuosoHandle>(null)
  const pendingFocus = useRef<string | null>(null)

  function focusPending() {
    if (!pendingFocus.current) return
    const button = document.getElementById(`skill-${pendingFocus.current}`)
    if (button) {
      button.focus()
      pendingFocus.current = null
    }
  }

  function move(event: React.KeyboardEvent, index: number) {
    if (!['ArrowDown', 'ArrowUp', 'Enter', ' '].includes(event.key)) return
    event.preventDefault()
    const next =
      event.key === 'ArrowDown'
        ? Math.min(props.groups.length - 1, index + 1)
        : event.key === 'ArrowUp'
          ? Math.max(0, index - 1)
          : index
    const group = props.groups[next]
    if (!group) return
    props.onOpen(group)
    pendingFocus.current = group.id
    list.current?.scrollToIndex({ index: next, align: 'center' })
    requestAnimationFrame(focusPending)
  }

  if (!props.groups.length)
    return <div className="skill-list">{props.empty}</div>
  return (
    <div
      className="skill-list"
      id="skills-panel"
      role="tabpanel"
      aria-label={t('Skills')}
      aria-busy={props.scanning}
    >
      <Virtuoso
        ref={list}
        data={props.groups}
        style={{ height: '100%' }}
        defaultItemHeight={68}
        computeItemKey={(_, group) => group.id}
        rangeChanged={() => requestAnimationFrame(focusPending)}
        itemContent={(index, group) => {
          const selected = group.members.some(
            (copy) => copy.id === props.selectedId,
          )
          const platforms = platformNames(group.tools).filter((name) =>
            props.agents.some(
              (agent) => agent.name === name && agent.installed,
            ),
          )
          const badges = platforms.length
            ? platforms
            : group.tools.includes('Shared')
              ? ['Shared']
              : ['Local']
          const label = platforms.length
            ? t('{count} platforms: {tools}', {
                count: platforms.length,
                tools: platforms.join(', '),
              })
            : t('Found in {tools}', {
                tools: badges
                  .map((name) =>
                    t(
                      name === 'Shared'
                        ? 'Shared Skills'
                        : name === 'Local'
                          ? 'Local files'
                          : name,
                    ),
                  )
                  .join(', '),
              })
          return (
            <div
              className={`skill-row${selected ? ' selected' : ''}`}
              role="listitem"
              aria-posinset={index + 1}
              aria-setsize={props.groups.length}
            >
              <button
                className="skill-select"
                id={`skill-${group.id}`}
                aria-label={t('Open {name}', { name: group.name })}
                aria-pressed={selected}
                disabled={!props.verified}
                onKeyDown={(event) => move(event, index)}
                onClick={() => props.onOpen(group)}
              >
                <FileText className="skill-glyph" size={16} />
                <div className="skill-info">
                  <span className="skill-name">
                    {group.name}
                    {group.versions.length > 1 && (
                      <small
                        className="version-label"
                        title={t('Same name, different Skill content')}
                      >
                        {t('{count} versions', {
                          count: group.versions.length,
                        })}
                      </small>
                    )}
                    {group.members.every((copy) =>
                      copy.occurrences?.every((item) => item.linked),
                    ) && <Link size={11} />}
                    {group.members.some(
                      (copy) => (copy.warnings?.length ?? 0) > 0,
                    ) && (
                      <AlertCircle
                        size={11}
                        aria-label={t('Metadata needs attention')}
                      />
                    )}
                  </span>
                  <span className="skill-description">
                    {group.description ||
                      t('Open the raw file to view or repair metadata.')}
                  </span>
                </div>
                <span
                  className="skill-origins"
                  title={label}
                  aria-label={label}
                >
                  <span className="logo-stack">
                    {badges.slice(0, 3).map((name) => (
                      <span className="platform-logo" key={name}>
                        <AgentIcon
                          name={name}
                          icon={
                            props.agents.find((agent) => agent.name === name)
                              ?.icon
                          }
                          size={14}
                        />
                      </span>
                    ))}
                  </span>
                  {platforms.length > 0 && (
                    <span className="platform-count">{platforms.length}</span>
                  )}
                </span>
              </button>
              <button
                className={`favorite${group.favorite ? ' saved' : ''}`}
                aria-label={t(
                  group.favorite ? 'Unfavorite {name}' : 'Favorite {name}',
                  { name: group.name },
                )}
                aria-pressed={group.favorite}
                disabled={!props.verified}
                onClick={() => props.onFavorite(group)}
              >
                <Star
                  size={13}
                  fill={group.favorite ? 'currentColor' : 'none'}
                />
              </button>
            </div>
          )
        }}
      />
    </div>
  )
}

export function AgentTabs({
  agents,
  groups,
  value,
  onChange,
}: {
  agents: AgentInfo[]
  groups: SkillGroup[]
  value: string
  onChange: (agent: string) => void
}) {
  const { t } = useLocale()
  const scroll = useRef<HTMLDivElement>(null)
  const [overflow, setOverflow] = useState(false)
  useEffect(() => {
    const element = scroll.current
    if (!element) return
    const measure = () =>
      setOverflow(element.scrollWidth > element.clientWidth + 1)
    const observer = new ResizeObserver(measure)
    observer.observe(element)
    measure()
    return () => observer.disconnect()
  }, [agents, groups, t])
  const tabs = [{ name: '', label: t('All'), icon: '' }]
  for (const agent of agents) {
    if (agent.installed)
      tabs.push({ name: agent.name, label: agent.name, icon: agent.icon })
  }
  for (const name of ['Shared', 'Local']) {
    if (groups.some((group) => group.tools.includes(name)))
      tabs.push({
        name,
        label: t(name === 'Shared' ? 'Shared Skills' : 'Local files'),
        icon: '',
      })
  }
  return (
    <div className="agent-tabs-shell">
      {overflow && (
        <IconButton
          icon={ChevronLeft}
          label={t('Scroll Agent tabs left')}
          onClick={() =>
            scroll.current?.scrollBy({ left: -220, behavior: 'smooth' })
          }
        />
      )}
      <div
        ref={scroll}
        className="agent-tabs"
        role="tablist"
        aria-label={t('Agents')}
      >
        {tabs.map((agent, index) => (
          <button
            key={agent.name}
            role="tab"
            aria-selected={value === agent.name}
            aria-controls="skills-panel"
            tabIndex={value === agent.name ? 0 : -1}
            onClick={() => onChange(agent.name)}
            onKeyDown={(event) => {
              if (
                !['ArrowLeft', 'ArrowRight', 'Home', 'End'].includes(event.key)
              )
                return
              event.preventDefault()
              const next =
                event.key === 'Home'
                  ? 0
                  : event.key === 'End'
                    ? tabs.length - 1
                    : (index +
                        (event.key === 'ArrowRight' ? 1 : -1) +
                        tabs.length) %
                      tabs.length
              onChange(tabs[next].name)
              const buttons =
                event.currentTarget.parentElement?.querySelectorAll<HTMLButtonElement>(
                  '[role="tab"]',
                )
              buttons?.[next].focus()
              buttons?.[next].scrollIntoView({
                block: 'nearest',
                inline: 'nearest',
              })
            }}
          >
            {agent.name && <AgentIcon name={agent.name} icon={agent.icon} />}
            {agent.label}
            <span>
              {agent.name
                ? groups.filter((group) => group.tools.includes(agent.name))
                    .length
                : groups.length}
            </span>
          </button>
        ))}
      </div>
      {overflow && (
        <IconButton
          icon={ChevronRight}
          label={t('Scroll Agent tabs right')}
          onClick={() =>
            scroll.current?.scrollBy({ left: 220, behavior: 'smooth' })
          }
        />
      )}
    </div>
  )
}
