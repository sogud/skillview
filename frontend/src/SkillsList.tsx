import { useRef, type ReactNode } from 'react'
import { FileText, BookOpen, Link, AlertCircle } from 'lucide-react'
import { Virtuoso, VirtuosoGrid, type VirtuosoHandle, type VirtuosoGridHandle } from 'react-virtuoso'
import type { AgentInfo } from './api'
import { AgentIcon } from './AgentIcon'
import { useLocale } from './locales'
import { platformNames, type SkillGroup } from './skill-groups'

type Props = {
  groups: SkillGroup[]
  agents: AgentInfo[]
  selectedId?: string
  verified: boolean
  scanning: boolean
  onOpen: (group: SkillGroup) => void
  layout: 'list' | 'shelf'
  onLocations: (group: SkillGroup) => void
  empty: ReactNode
}

export function SkillsList(props: Props) {
  const { t, direction } = useLocale()
  const list = useRef<VirtuosoHandle>(null)
  const grid = useRef<VirtuosoGridHandle>(null)
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
    if (!['ArrowDown', 'ArrowUp', 'ArrowLeft', 'ArrowRight', 'Enter', ' '].includes(event.key)) return
    event.preventDefault()
    const next =
      event.key === 'ArrowDown' || (event.key === (direction === 'rtl' ? 'ArrowLeft' : 'ArrowRight'))
        ? Math.min(props.groups.length - 1, index + 1)
        : event.key === 'ArrowUp' || event.key === (direction === 'rtl' ? 'ArrowRight' : 'ArrowLeft')
          ? Math.max(0, index - 1)
          : index
    const group = props.groups[next]
    if (!group) return
    if (event.key === 'Enter' || event.key === ' ') {
      props.onOpen(group)
      return
    }
    pendingFocus.current = group.id
    list.current?.scrollToIndex({ index: next, align: 'center' })
    grid.current?.scrollToIndex({ index: next, align: 'center' })
    requestAnimationFrame(focusPending)
  }

  function item(index: number, group: SkillGroup) {
    let locationCount = 0
    for (const copy of group.members) {
      locationCount += new Set((copy.occurrences ?? []).map((entry) => entry.path)).size
    }
  const hasWarnings = group.members.some((copy) => (copy.warnings?.length ?? 0) > 0)
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
                ? 'Common Skills'
                : name === 'Local'
                  ? 'Local files'
                  : name,
            ),
          )
          .join(', '),
      })
  const copyLabels: string[] = []
  if (group.members.length > 1) copyLabels.push(t('{count} copies', { count: group.members.length }))
  if (group.versions.length > 1) copyLabels.push(t('{count} versions', { count: group.versions.length }))
  const copyLabel = copyLabels.join(' · ')
  const origins = (
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
  )
  const locationsLabel = t('{count} discovered locations', { count: locationCount })
  const locations = (
    <button className="skill-locations" disabled={!props.verified} onClick={() => props.onLocations(group)}
      aria-label={`${t('Locations and references')}: ${locationsLabel}`} title={`${t('Locations and references')}: ${locationsLabel}`}>
      <Link size={13} aria-hidden="true" /><span>{props.layout === 'shelf' ? locationCount : locationsLabel}</span>
    </button>
  )
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
        aria-description={props.layout === 'shelf' && hasWarnings ? t('Metadata needs attention') : undefined}
        aria-pressed={selected}
        disabled={!props.verified}
        onKeyDown={(event) => move(event, index)}
        onClick={() => props.onOpen(group)}
      >
        {props.layout === 'shelf' ? <BookOpen className="skill-glyph" size={16} aria-hidden="true" /> : <FileText className="skill-glyph" size={16} />}
        <div className="skill-info">
          <span className="skill-name">
            <span className="skill-title" dir="auto" title={group.name}>
              {group.name}
            </span>
            {props.layout === 'list' && group.members.length > 1 && <small className="version-label">{t('{count} copies', { count: group.members.length })}</small>}
            {props.layout === 'list' && group.versions.length > 1 && (
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
            {props.layout === 'list' && hasWarnings && (
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
        {props.layout === 'list' && origins}
      </button>
      {props.layout === 'shelf' ? (
        <div className="skill-card-footer">
          <span className="skill-card-meta">
            {origins}
            {copyLabel && <span className="skill-card-counts" title={copyLabel}>{copyLabel}</span>}
          </span>
          {locations}
        </div>
      ) : locations}
    </div>
  )
  }
  if (!props.groups.length) return <div className="skill-list">{props.empty}</div>
  return <div className={`skill-list skills-layout ${props.layout}`} id="skills-panel" role="tabpanel" aria-label={t('Skills')} aria-busy={props.scanning}>
    {props.layout === 'shelf' ? <VirtuosoGrid ref={grid} data={props.groups} style={{ height: '100%' }} listClassName="skill-shelf" itemClassName="skill-shelf-item" computeItemKey={(_, group) => group.id} rangeChanged={() => requestAnimationFrame(focusPending)} itemContent={item} /> : <Virtuoso ref={list} data={props.groups} style={{ height: '100%' }} defaultItemHeight={64} computeItemKey={(_, group) => group.id} rangeChanged={() => requestAnimationFrame(focusPending)} itemContent={item} />}
  </div>
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
  const { t, direction } = useLocale()
  const tabs = [{ name: '', label: t('All'), icon: '' }]
  for (const agent of agents) {
    if (agent.installed)
      tabs.push({ name: agent.name, label: agent.name, icon: agent.icon })
  }
  for (const name of ['Shared', 'Local']) {
    if (groups.some((group) => group.tools.includes(name)))
      tabs.push({
        name,
        label: t(name === 'Shared' ? 'Common Skills' : 'Local files'),
        icon: '',
      })
  }
  return (
    <div className="agent-tabs" role="tablist" aria-label={t('Agents')}>
      {tabs.map((agent, index) => (
        <button
          key={agent.name}
          role="tab"
          aria-selected={value === agent.name}
          aria-controls="skills-panel"
          tabIndex={value === agent.name ? 0 : -1}
          onClick={() => onChange(agent.name)}
          onKeyDown={(event) => {
            if (!['ArrowLeft', 'ArrowRight', 'Home', 'End'].includes(event.key))
              return
            event.preventDefault()
            const next =
              event.key === 'Home'
                ? 0
                : event.key === 'End'
                  ? tabs.length - 1
                  : (index +
                      (event.key === 'ArrowRight' ? 1 : -1) * (direction === 'rtl' ? -1 : 1) +
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
  )
}
