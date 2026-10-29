# Feedback and optional error reports

## Manual feedback

Choose **Send feedback**, describe the problem, review the locally redacted payload, then choose **Send reviewed report**. Feedback goes to `https://feedback.sogud.me`. Opening this dialog never submits anything. Automatic error reporting is a separate, off-by-default setting described below. Neither path records sessions, screenshots, behavior events or logs, and neither uses a disk queue.

Reports contain the entered description, app version, OS and architecture. The translation error row has a **Report this issue** action. An AI response format failure can optionally accompany feedback as a linked error, code `SV_AI_RESPONSE_FORMAT`. Fixed categories are `invalid_format`, `no_choices` and `incomplete_translations`; `finish_reason` is allowlisted. `truncation=reported` means the provider returned `length`; `not_reported` means a recognized other reason; otherwise it is `unknown`. None asserts that output was complete. The diagnostic contains no model/provider identifiers, prompts, documents or response text, stays in memory and clears when another translation starts.

Common token formats, credential assignments, URLs, local paths and email addresses are redacted before preview. Pattern-based redaction cannot identify arbitrary confidential prose: do not paste private documents or secrets. The native transport rejects SDK-added user data, breadcrumbs, traces, request URLs, extra fields and attachments. Only the reviewed description and fixed diagnostic fields are allowed.

## Sentry SDK and server contract

The pinned MIT `@sentry/browser` and `@sentry/core` 8.55.0 SDKs use an isolated BrowserClient/Scope, `captureFeedback`, optional `captureEvent`, and the official envelope serializer. The SDK is loaded only when serializing a manual submission or an opted-in error, with no global SDK init or default integrations. A custom native Wails transport avoids browser CORS changes. It supports only these restricted event/feedback subsets, not Sentry tracing, native crashes, symbolication or arbitrary SDK payloads.

One general report sends a `feedback` item. An AI report first sends one fixed `event` exception, then a `feedback` item linked by `associated_event_id`. OS/architecture are appended to the feedback message from the reviewed diagnostic fields. Error values contain only category/finish reason/truncation. Each item goes to `POST /api/1/envelope/`; the bundled DSN contains only the existing public ingestion key, with no administrator credential. An explicit `SKILLVIEW_FEEDBACK_DSN` overrides it for deployment/testing; an empty value disables sending. HTTPS is required except loopback test endpoints. Redirects are rejected.

Only HTTP 200 with a bounded, valid `{accepted, duplicates, skipped}` receipt confirming exactly one accepted/duplicate item and zero skipped items counts as confirmed delivery. This backend commits SQLite before acknowledging. SDK flush confirms local serialization only, never delivery. The backend currently has one configured project; the DSN selects it. No behavior capture is enabled by this integration.

## Manual delivery and failure isolation

At most one report is in flight, with no waiting queue, at most two requests and a six-second total deadline. There are zero automatic retries. Closing the feedback window cancels its request and never blocks reading, translation or application exit. DNS/connection/TLS failures, timeout, 429, 5xx, redirects and malformed receipts only produce a feedback error. A 429 imposes an in-memory cooldown, normally 60 seconds. No raw server body or network details are displayed.

Manual retries keep the reviewed IDs and timestamp; the backend deduplicates identical normalized events. This also recovers a partially delivered AI report. A timeout has unknown delivery outcome. Closing the dialog discards its in-memory draft; editing/reviewing creates a new report. Reports are synthetic in automated acceptance tests; production-network acceptance is opt-in.

## Automatic error reports

**Settings → General → Automatic error reports** is off for new and existing settings. Checking it explicitly agrees to send the described limited data to `https://feedback.sogud.me`; the choice is saved locally and can be disabled at any time. This setting is available from 0.1.11.

After saved consent loads, window listeners capture uncaught frontend exceptions and unhandled Promise rejections. Reports contain an allowlisted error type, app version, timestamp, random event ID and at most eight app stack locations. Only bundled app locations are recognized; every filename becomes `frontend.js`, with numeric line/column positions. Original messages, function names, source text, local paths, URL parameters, arbitrary rejected values, documents and user input are omitted rather than pattern-redacted. Native code reconstructs this exact allowlist and rejects extra SDK fields before sending. No administrator secret is included.

Sending is asynchronous. Frontend and native limits allow at most one attempt per minute and ten distinct errors per process; duplicate fingerprints are suppressed, including after a failed attempt. Automatic reports share the manual transport's single in-flight slot, have a three-second deadline and never retry or enter a waiting/disk queue. Disabling cancels an active automatic request; a request already accepted by the server cannot be recalled. Sender failures are swallowed, never recursively reported. A failed report does not change reading, saving or translation state.

This does not capture Go panics/native crashes, handled errors, failures before consent has loaded, or pure UI logic bugs that throw no exception. App locations are minified and are not symbolicated. It does not enable general logs or behavior analytics. Native settings toggling/restart persistence, one real synthetic WebKit → Wails → server delivery, no send after disabling, and normal reading/settings after a refused connection have been verified. The temporary trigger was removed and is absent from the clean build. Other network failure types retain shared-transport test coverage rather than new native pointer acceptance; see [verification](verification.md#automatic-errors-native-acceptance--2026-10-29-0834-utc--codex).
