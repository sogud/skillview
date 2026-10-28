// Resolve a Markdown link against the open file, never above its catalog root.
// File reads still go through the backend's identity and no-symlink checks.
export function resolveLocalLink(
  currentPath: string,
  href: string,
): string | null {
  const path = decodeURIComponent(href.split(/[?#]/)[0])
  if (/^(?:\/|[a-z][a-z\d+.-]*:)/i.test(path) || /[\\\0]/.test(path))
    return null
  if (!path) return currentPath
  const parts = currentPath.split('/').slice(0, -1)
  for (const part of path.split('/')) {
    if (!part || part === '.') continue
    if (part === '..') {
      if (!parts.length) return null
      parts.pop()
    } else parts.push(part)
  }
  return parts.length ? parts.join('/') : null
}
