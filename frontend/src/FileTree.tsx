import { useEffect, useMemo, useRef, useState } from 'react'
import {
  ChevronRight,
  Folder,
  FolderOpen,
  FileText,
  FileCode,
  Image,
  Link,
  FilePenLine,
  Trash2,
  ListTree,
  Plus,
  Upload,
} from 'lucide-react'
import type { Resource } from './api'
import { IconButton } from './components'
import { useLocale } from './locales'
import { buildFileTree, parentFolders, visibleTreeRows } from './file-tree'

export function FileTree({
  resources,
  path,
  editable,
  onOpen,
  onRename,
  onTrash,
  onNew,
  onImport,
}: {
  resources: Resource[]
  path: string
  editable: boolean
  onOpen: (path: string) => void
  onRename: (resource: Resource) => void
  onTrash: (resource: Resource) => void
  onNew: () => void
  onImport: () => void
}) {
  const { t } = useLocale()
  const nodes = useMemo(() => buildFileTree(resources), [resources])
  const [expanded, setExpanded] = useState(
    () => new Set(['references', 'scripts', 'assets']),
  )
  const [focused, setFocused] = useState(path)
  const tree = useRef<HTMLDivElement>(null)
  useEffect(() => {
    setExpanded((previous) => new Set([...previous, ...parentFolders(path)]))
    setFocused(path)
  }, [path])
  const rows = visibleTreeRows(nodes, expanded)
  function toggle(folder: string) {
    setExpanded((previous) => {
      const next = new Set(previous)
      if (next.has(folder)) next.delete(folder)
      else next.add(folder)
      return next
    })
  }
  function focus(index: number) {
    if (!rows[index]) return
    setFocused(rows[index].node.path)
    tree.current
      ?.querySelectorAll<HTMLButtonElement>('[role="treeitem"]')
      [index]?.focus()
  }
  return (
    <>
      <div className="explorer-label">
        <span>{t('Files')}</span>
        <div className="explorer-tools">
          {editable && (
            <>
              <IconButton
                icon={Plus}
                label={t('New resource')}
                onClick={onNew}
              />
              <IconButton
                icon={Upload}
                label={t('Import resource')}
                onClick={onImport}
              />
            </>
          )}
          <IconButton
            icon={ListTree}
            label={t('Collapse folders')}
            onClick={() => {
              setExpanded(new Set())
              setFocused(nodes[0]?.path ?? '')
            }}
          />
        </div>
      </div>
      <div
        ref={tree}
        className="file-tree"
        role="tree"
        aria-label={t('Skill files')}
      >
        {rows.map(({ node, depth }, index) => {
          const resource = node.resource
          const folder = !resource
          const linked = resource?.kind === 'link'
          const Icon = folder
            ? expanded.has(node.path)
              ? FolderOpen
              : Folder
            : linked
              ? Link
              : resource.kind === 'image'
                ? Image
                : /\.(sh|py|js|ts|json|yaml|yml|toml|go|rs)$/.test(node.name)
                  ? FileCode
                  : FileText
          return (
            <div
              className={`tree-row${path === node.path ? ' selected' : ''}`}
              key={node.path}
            >
              <button
                type="button"
                role="treeitem"
                aria-level={depth + 1}
                aria-expanded={folder ? expanded.has(node.path) : undefined}
                aria-selected={!folder ? path === node.path : undefined}
                aria-disabled={linked || undefined}
                tabIndex={
                  focused === node.path ||
                  (!rows.some((row) => row.node.path === focused) &&
                    index === 0)
                    ? 0
                    : -1
                }
                style={{ paddingLeft: 8 + depth * 14 }}
                title={
                  linked ? t('Resource links are not followed') : node.path
                }
                onFocus={() => setFocused(node.path)}
                onClick={() => {
                  if (folder) toggle(node.path)
                  else if (!linked) onOpen(node.path)
                }}
                onKeyDown={(event) => {
                  const key = event.key
                  if (
                    ![
                      'ArrowDown',
                      'ArrowUp',
                      'ArrowLeft',
                      'ArrowRight',
                      'Home',
                      'End',
                    ].includes(key)
                  )
                    return
                  event.preventDefault()
                  if (key === 'ArrowDown') focus(index + 1)
                  else if (key === 'ArrowUp') focus(index - 1)
                  else if (key === 'Home') focus(0)
                  else if (key === 'End') focus(rows.length - 1)
                  else if (key === 'ArrowRight' && folder) {
                    if (!expanded.has(node.path)) toggle(node.path)
                    else focus(index + 1)
                  } else if (key === 'ArrowLeft') {
                    if (folder && expanded.has(node.path)) toggle(node.path)
                    else {
                      const parent = parentFolders(node.path).at(-1)
                      focus(rows.findIndex((row) => row.node.path === parent))
                    }
                  }
                }}
              >
                {folder ? (
                  <ChevronRight
                    size={12}
                    className={
                      expanded.has(node.path)
                        ? 'tree-chevron open'
                        : 'tree-chevron'
                    }
                  />
                ) : (
                  <span className="tree-chevron" />
                )}
                <Icon size={14} />
                <span>{node.name}</span>
              </button>
              {editable && resource && resource.path !== 'SKILL.md' && (
                <div className="tree-actions">
                  {!linked && (
                    <IconButton
                      icon={FilePenLine}
                      label={t('Rename {name}', { name: resource.path })}
                      onClick={() => onRename(resource)}
                    />
                  )}
                  <IconButton
                    icon={Trash2}
                    label={
                      linked
                        ? t('Remove link ({path})', { path: resource.path })
                        : t('Move {name} to Trash', { name: resource.path })
                    }
                    onClick={() => onTrash(resource)}
                  />
                </div>
              )}
            </div>
          )
        })}
      </div>
    </>
  )
}
