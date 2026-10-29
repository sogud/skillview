import { BrowserClient, Scope, captureFeedback } from '@sentry/browser'
import {
  serializeEnvelope,
  type Event,
  type Envelope,
  type EventEnvelope,
} from '@sentry/core'

export type ReviewedFeedback = {
  event_id: string
  error_event_id?: string
  timestamp: string
  release: string
  message: string
  diagnostics: {
    os: string
    arch: string
    ai_response?: {
      category: string
      finish_reason: string
      truncation: string
    } | null
  }
}

// No global init, default integrations, session tracking, tracing, replay,
// client reports, network transport or disk queue.
// Loaded for reviewed manual reports or explicitly enabled automatic errors.
export async function buildFeedbackEnvelopes(
  dsn: string,
  report: ReviewedFeedback,
): Promise<string[]> {
  const wires: string[] = []
  const common = {
    timestamp: Date.parse(report.timestamp) / 1000,
    release: report.release,
    environment: 'production',
    platform: 'javascript',
  }
  const client = new BrowserClient({
    dsn,
    integrations: [],
    stackParser: () => [],
    sendClientReports: false,
    sendDefaultPii: false,
    sampleRate: 1,
    tracesSampleRate: 0,
    transport: () => ({
      send(envelope: Envelope) {
        if (wires.length >= 2 || envelope[1].length !== 1)
          throw new Error('Could not prepare feedback.')
        const [header, input] = envelope[1][0]
        if (header.type !== 'event' && header.type !== 'feedback')
          throw new Error('Could not prepare feedback.')
        const event = input as Event
        // Drop SDK-added trace IDs, scope metadata and all ambient browser data.
        const clean: Event = {
          event_id: event.event_id,
          ...common,
          level: event.level,
        }
        if (header.type === 'feedback') {
          clean.type = 'feedback'
          const feedback = event.contexts?.feedback
          clean.contexts = {
            feedback: {
              message: feedback?.message,
              ...(feedback?.associated_event_id
                ? { associated_event_id: feedback.associated_event_id }
                : {}),
            },
          }
        } else {
          clean.exception = event.exception
        }
        if (!clean.event_id) throw new Error('Could not prepare feedback.')
        const filtered: EventEnvelope = [
          { event_id: clean.event_id, sent_at: report.timestamp },
          [[{ type: header.type }, clean]],
        ]
        const wire = serializeEnvelope(filtered)
        if (typeof wire !== 'string' || wire.length > 32768)
          throw new Error('Could not prepare feedback.')
        wires.push(wire)
        return Promise.resolve({ statusCode: 200 }) // serialization only; never a delivery receipt
      },
      flush: () => Promise.resolve(true),
    }),
  })
  const scope = new Scope()
  scope.setClient(client)
  client.init()
  try {
    const d = report.diagnostics.ai_response
    if (d)
      scope.captureEvent({
        event_id: report.error_event_id,
        ...common,
        level: 'error',
        exception: {
          values: [
            {
              type: 'SV_AI_RESPONSE_FORMAT',
              value: `category=${d.category}; finish_reason=${d.finish_reason}; truncation=${d.truncation}`,
            },
          ],
        },
      })
    captureFeedback(
      {
        message: `${report.message}\n\nOS: ${report.diagnostics.os}; Arch: ${report.diagnostics.arch}`,
        ...(report.error_event_id
          ? { associatedEventId: report.error_event_id }
          : {}),
      },
      { event_id: report.event_id },
      scope,
    )
    if (!(await client.flush(1000)) || wires.length !== (d ? 2 : 1))
      throw new Error('Could not prepare feedback.')
    return wires
  } finally {
    await client.close(1000)
  }
}

export type AutomaticErrorReport = {
  event_id: string
  timestamp: string
  release: string
  environment: string
  source: string
  error_type: string
  frames: { filename: string; lineno: number; colno: number }[]
}

// The native sender independently reconstructs and compares this exact whitelist.
export async function buildAutomaticErrorEnvelope(
  dsn: string,
  report: AutomaticErrorReport,
): Promise<string> {
  const event: Event = {
    event_id: report.event_id,
    timestamp: Date.parse(report.timestamp) / 1000,
    release: report.release,
    environment: report.environment,
    platform: 'javascript',
    level: 'error',
    exception: { values: [{
      type: report.error_type,
      value: 'Automatic error report; message omitted.',
      ...(report.frames.length ? { stacktrace: { frames: report.frames.map((frame) => ({ ...frame, in_app: true })) } } : {}),
    }] },
    tags: {
      source: 'automatic_error',
      error_origin: report.source,
      ...(report.environment === 'synthetic' ? { synthetic: 'true' } : {}),
    },
    fingerprint: ['skillview-auto', report.source, report.error_type, ...report.frames.map((frame) => `${frame.lineno}:${frame.colno}`)],
  }
  // Snapshot before Sentry can add ambient scope or runtime metadata.
  const cleanEvent = structuredClone(event)
  let wire = ''
  const client = new BrowserClient({
    dsn, integrations: [], stackParser: () => [],
    sendClientReports: false, sendDefaultPii: false,
    sampleRate: 1, tracesSampleRate: 0,
    transport: () => ({
      send(envelope: Envelope) {
        if (wire || envelope[1].length !== 1 || envelope[1][0][0].type !== 'event')
          throw new Error('Could not prepare feedback.')
        const clean: EventEnvelope = [
          { event_id: report.event_id, sent_at: report.timestamp },
          [[{ type: 'event' }, cleanEvent]],
        ]
        const serialized = serializeEnvelope(clean)
        if (typeof serialized !== 'string' || serialized.length > 32768)
          throw new Error('Could not prepare feedback.')
        wire = serialized
        return Promise.resolve({ statusCode: 200 })
      },
      flush: () => Promise.resolve(true),
    }),
  })
  const scope = new Scope()
  scope.setClient(client)
  client.init()
  try {
    scope.captureEvent(event)
    if (!(await client.flush(1000)) || !wire) throw new Error('Could not prepare feedback.')
    return wire
  } finally {
    await client.close(1000)
  }
}
