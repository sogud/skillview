import assert from 'node:assert/strict'
import test from 'node:test'
import { groupSkills, selectCopy, platformNames } from './skill-groups.ts'

const copy = (id, contentKey, tools, editable = false) => ({ id, contentKey, name: 'sample', description: 'A sample', tools, editable, favorite: false })

test('one description combines matching copies and unique platform membership', () => {
  const groups = groupSkills([copy('a', 'same', ['Codex']), copy('b', 'same', ['Claude Code', 'Codex']), copy('c', 'same', ['Local'])])
  assert.equal(groups.length, 1)
  assert.equal(groups[0].members.length, 3)
  assert.deepEqual(platformNames(groups[0].tools), ['Codex', 'Claude Code'])
  assert.equal(selectCopy(groups[0], 'Claude Code').id, 'b')
})

test('same-name versions occupy one row while retaining different content and copies', () => {
  const groups = groupSkills([copy('a', 'one', ['Codex']), copy('b', 'two', ['Claude Code'])])
  assert.equal(groups.length, 1)
  assert.deepEqual(groups[0].versions, ['one', 'two'])
  assert.equal(groups[0].members.length, 2)
  const other = { ...copy('c', 'one', ['Codex']), name: 'another-skill' }
  assert.equal(groupSkills([groups[0].members[0], other]).length, 2)
})

test('each platform opens its own copy without inheriting another copy’s write access', () => {
  const group = groupSkills([copy('read', 'same', ['Codex']), copy('write', 'same', ['Local'], true)])[0]
  assert.equal(selectCopy(group).id, 'write')
  assert.equal(selectCopy(group, 'Codex').editable, false)
  assert.equal(selectCopy(group, '', 'read').id, 'read')
})
