import test from 'node:test'
import assert from 'node:assert/strict'
import { buildFileTree, parentFolders, visibleTreeRows } from './file-tree.ts'

test('nested resources appear once with folder names, main document first, and links as leaves', () => {
  const resources = ['references/sheet/read.md', 'SKILL.md', 'scripts/run.py', 'references/sheet/write.md', 'shortcut'].map((path) => ({ path, kind: path === 'shortcut' ? 'link' : 'text', size: 1 }))
  const tree = buildFileTree(resources)
  assert.deepEqual(tree.map((node) => node.name), ['SKILL.md', 'references', 'scripts', 'shortcut'])
  assert.deepEqual(visibleTreeRows(tree, new Set(['references', 'references/sheet'])).map(({node}) => node.name), ['SKILL.md', 'references', 'sheet', 'read.md', 'write.md', 'scripts', 'shortcut'])
  assert.equal(tree.at(-1).resource.kind, 'link')
  assert.deepEqual(parentFolders('references/sheet/read.md'), ['references', 'references/sheet'])
})

test('collapsed folders hide children without losing their resource identity', () => {
  const resource = { path: 'references/guide.md', kind: 'markdown', revision: 'r', size: 20 }
  const tree = buildFileTree([resource])
  assert.equal(visibleTreeRows(tree, new Set()).length, 1)
  assert.equal(visibleTreeRows(tree, new Set(['references']))[1].node.resource, resource)
})
