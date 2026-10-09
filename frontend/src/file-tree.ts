import type { Resource } from './api'

export type FileNode = {
  name: string
  path: string
  children: FileNode[]
  resource?: Resource
}
export type TreeRow = { node: FileNode; depth: number }

export function buildFileTree(resources: Resource[]): FileNode[] {
  const roots: FileNode[] = []
  for (const resource of resources) {
    let children = roots
    const parts = resource.path.split('/')
    let path = ''
    for (let index = 0; index < parts.length; index++) {
      const name = parts[index]
      path = path ? `${path}/${name}` : name
      let node = children.find((item) => item.name === name)
      if (!node) {
        node = { name, path, children: [] }
        children.push(node)
      }
      if (index === parts.length - 1) node.resource = resource
      children = node.children
    }
  }
  function sort(nodes: FileNode[]) {
    nodes.sort((a, b) => {
      if (a.path === 'SKILL.md') return -1
      if (b.path === 'SKILL.md') return 1
      if (!!a.resource !== !!b.resource) return a.resource ? 1 : -1
      return a.name.localeCompare(b.name)
    })
    for (const node of nodes) sort(node.children)
  }
  sort(roots)
  return roots
}

export function visibleTreeRows(
  nodes: FileNode[],
  expanded: Set<string>,
): TreeRow[] {
  const rows: TreeRow[] = []
  function visit(children: FileNode[], depth: number) {
    for (const node of children) {
      rows.push({ node, depth })
      if (!node.resource && expanded.has(node.path))
        visit(node.children, depth + 1)
    }
  }
  visit(nodes, 0)
  return rows
}

export function parentFolders(path: string): string[] {
  const folders: string[] = []
  const parts = path.split('/')
  for (let index = 1; index < parts.length; index++)
    folders.push(parts.slice(0, index).join('/'))
  return folders
}
