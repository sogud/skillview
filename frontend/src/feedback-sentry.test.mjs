import test from 'node:test'
import assert from 'node:assert/strict'
import { getClient, getGlobalScope } from '@sentry/browser'
import { buildFeedbackEnvelopes } from './feedback-sentry.ts'
const report = {
  event_id: '1'.repeat(32),
  timestamp: '2026-10-28T09:00:00Z',
  release: '0.1.7',
  message: 'Synthetic reviewed feedback',
  diagnostics: { os: 'darwin', arch: 'arm64' },
}
const dsn = `https://${'a'.repeat(32)}@feedback.invalid/1`
test('actual Sentry captureFeedback uses stable IDs, no network/global client or ambient private data', async () => {
  const original = getClient()
  const global = getGlobalScope()
  global.setUser({ email: 'private@example.invalid' })
  global.setExtra('document', 'PRIVATE_DOCUMENT')
  global.addBreadcrumb({ message: 'PRIVATE_RESPONSE' })
  try {
    const first = await buildFeedbackEnvelopes(dsn, report)
    assert.deepEqual(await buildFeedbackEnvelopes(dsn, report), first)
    assert.equal(first.length, 1)
    const [h, i, e] = first[0].split('\n').map(JSON.parse)
    assert.deepEqual(h, {
      event_id: report.event_id,
      sent_at: report.timestamp,
    })
    assert.deepEqual(i, { type: 'feedback' })
    assert.equal(
      e.contexts.feedback.message,
      report.message + '\n\nOS: darwin; Arch: arm64',
    )
    assert.equal(e.release, '0.1.7')
    assert.equal(e.timestamp, Date.parse(report.timestamp) / 1000)
    assert.ok(!first[0].includes('PRIVATE_'))
    assert.ok(!first[0].includes('private@example'))
    assert.deepEqual(Object.keys(e.contexts), ['feedback'])
    assert.equal(e.user, undefined)
    assert.equal(e.breadcrumbs, undefined)
    assert.equal(getClient(), original)
  } finally {
    global.clear()
  }
})
test('AI report creates one fixed error and one linked feedback, stable over partial retries', async () => {
  const p = {
    ...report,
    error_event_id: '2'.repeat(32),
    diagnostics: {
      ...report.diagnostics,
      ai_response: {
        category: 'invalid_format',
        finish_reason: 'length',
        truncation: 'reported',
      },
    },
  }
  const wires = await buildFeedbackEnvelopes(dsn, p)
  assert.equal(wires.length, 2)
  assert.deepEqual(await buildFeedbackEnvelopes(dsn, p), wires)
  const e = wires[0].split('\n').map(JSON.parse)[2],
    f = wires[1].split('\n').map(JSON.parse)[2]
  assert.equal(e.exception.values[0].type, 'SV_AI_RESPONSE_FORMAT')
  assert.equal(
    e.exception.values[0].value,
    'category=invalid_format; finish_reason=length; truncation=reported',
  )
  assert.equal(e.exception.values[0].stacktrace, undefined)
  assert.equal(f.contexts.feedback.associated_event_id, e.event_id)
})
