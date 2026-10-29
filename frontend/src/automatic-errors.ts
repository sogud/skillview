import type { AutomaticErrorReport } from './feedback-sentry'

const errorTypes = new Set(['Error', 'TypeError', 'RangeError', 'ReferenceError', 'SyntaxError', 'URIError', 'EvalError', 'AggregateError'])
// Only bundled app code is recognized. Uploaded fields contain no URL or function name.
const appLocation = /(?:^|[@ (])(wails:\/\/localhost\/assets\/(?:index|Editor)-[A-Za-z0-9_-]{1,64}\.js):(\d+):(\d+)(?:\)?$)/
const appFile = /^wails:\/\/localhost\/assets\/(?:index|Editor)-[A-Za-z0-9_-]{1,64}\.js$/

export function describeAutomaticError(reason: unknown, source: string, filename = '', line = 0, column = 0) {
  let errorType = source === 'unhandled_rejection' ? 'NonErrorRejection' : 'Error'
  let stack = ''
  if (reason instanceof Error) {
    errorType = 'Error'
    if (errorTypes.has(reason.name)) errorType = reason.name
    if (typeof reason.stack === 'string') stack = reason.stack.slice(0, 16384)
  }
  // Do not report failures inside the reporter itself.
  if (/\/assets\/(?:automatic-errors|feedback-sentry)-/.test(stack)) return null
  const frames: AutomaticErrorReport['frames'] = []
  const add = (lineno: number, colno: number) => {
    if (Number.isInteger(lineno) && lineno > 0 && lineno <= 1000000 && Number.isInteger(colno) && colno > 0 && colno <= 10000000)
      frames.push({ filename: 'frontend.js', lineno, colno })
  }
  for (const text of stack.split('\n').slice(0, 32)) {
    const location = text.match(appLocation)
    if (location) add(Number(location[2]), Number(location[3]))
    if (frames.length === 8) break
  }
  if (!frames.length && appFile.test(filename)) add(line, column)
  frames.reverse()
  return { source, error_type: errorType, frames }
}

type Sending = Promise<unknown> & { cancel?: () => void }

export function startAutomaticErrors(
  config: { dsn: string; release: string; environment: string },
  send: (report: AutomaticErrorReport, envelope: string) => Sending,
): () => void {
  let stopped = false
  let busy = false
  let last = 0
  let pending: Sending | null = null
  const seen = new Set<string>()
  async function capture(reason: unknown, source: string, filename = '', line = 0, column = 0) {
    if (stopped || busy || seen.size >= 10 || Date.now() - last < 60000) return
    try {
      const details = describeAutomaticError(reason, source, filename, line, column)
      if (!details) return
      const identity = JSON.stringify(details)
      if (seen.has(identity)) return
      seen.add(identity)
      last = Date.now()
      busy = true
      const bytes = crypto.getRandomValues(new Uint8Array(16))
      const report: AutomaticErrorReport = {
        ...details,
        event_id: Array.from(bytes, (byte) => byte.toString(16).padStart(2, '0')).join(''),
        timestamp: new Date(Math.floor(Date.now() / 1000) * 1000).toISOString(),
        release: config.release,
        environment: config.environment,
      }
      const { buildAutomaticErrorEnvelope } = await import('./feedback-sentry')
      const envelope = await buildAutomaticErrorEnvelope(config.dsn, report)
      if (stopped) return
      pending = send(report, envelope)
      await pending
    } catch {
      // Never report the reporter's failure, show a global error, or retry.
    } finally {
      busy = false
      pending = null
    }
  }
  const onError = (event: ErrorEvent) => { void capture(event.error, 'uncaught_exception', event.filename, event.lineno, event.colno) }
  const onRejection = (event: PromiseRejectionEvent) => { void capture(event.reason, 'unhandled_rejection') }
  window.addEventListener('error', onError)
  window.addEventListener('unhandledrejection', onRejection)
  function stop() {
    stopped = true
    window.removeEventListener('error', onError)
    window.removeEventListener('unhandledrejection', onRejection)
    window.removeEventListener('pagehide', stop)
    pending?.cancel?.()
  }
  window.addEventListener('pagehide', stop)
  return stop
}

let reportingGeneration = 0
let stopReporting: (() => void) | undefined

// Window lifetime, not React component lifetime: a rendering failure can unmount App.
export function configureAutomaticErrors(
  enabled: boolean,
  configuration: () => Promise<{ enabled: boolean; automaticErrors: boolean; dsn: string; release: string; environment: string }>,
  send: (report: AutomaticErrorReport, envelope: string) => Sending,
) {
  const generation = ++reportingGeneration
  stopReporting?.()
  stopReporting = undefined
  if (!enabled) return
  void configuration().then((config) => {
    if (generation === reportingGeneration && config.enabled && config.automaticErrors)
      stopReporting = startAutomaticErrors(config, send)
  }).catch(() => {})
}
