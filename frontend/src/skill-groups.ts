import type { Skill } from './api'

export type SkillGroup = {
  id: string
  name: string
  description: string
  members: Skill[]
  tools: string[]
  favorite: boolean
  versions: string[]
}

export function groupSkills(skills: Skill[]): SkillGroup[] {
  const groups: SkillGroup[] = []
  const byName = new Map<string, SkillGroup>()
  for (const skill of skills) {
    const key = skill.name.trim().toLowerCase() || skill.id
    let group = byName.get(key)
    if (!group) {
      group = {
        id: key,
        name: skill.name,
        description: skill.description,
        members: [],
        tools: [],
        favorite: false,
        versions: [],
      }
      byName.set(key, group)
      groups.push(group)
    }
    group.members.push(skill)
    if (!group.description && skill.description)
      group.description = skill.description
    const version = skill.contentKey || skill.revision || skill.id
    if (!group.versions.includes(version)) group.versions.push(version)
    group.favorite ||= skill.favorite
    for (const tool of skill.tools ?? []) {
      if (!group.tools.includes(tool)) group.tools.push(tool)
    }
  }
  return groups
}

export function selectCopy(
  group: SkillGroup,
  agent = '',
  selectedId = '',
): Skill {
  const eligible = agent
    ? group.members.filter((skill) => skill.tools?.includes(agent))
    : group.members
  const copies = eligible.length ? eligible : group.members
  return (
    copies.find((skill) => skill.id === selectedId) ??
    copies.find((skill) => skill.editable) ??
    copies[0]
  )
}

export function platformNames(tools: string[]): string[] {
  return tools.filter((tool) => tool !== 'Shared' && tool !== 'Local')
}
