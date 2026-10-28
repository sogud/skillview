# Manual feedback

Choose **Send feedback**, describe the problem, review the locally redacted payload, then choose **Send reviewed report**. Feedback goes to `https://feedback.sogud.me`. Nothing is uploaded at startup, while reading or translating, or simply by opening this dialog. There is no automatic telemetry, session recording, screenshot, log upload or disk queue.

Reports contain the entered description, app version, OS and architecture. The translation error row has a **Report this issue** action. An AI response format failure can optionally accompany feedback as a linked error, code `SV_AI_RESPONSE_FORMAT`. Fixed categories are `invalid_format`, `no_choices` and `incomplete_translations`; `finish_reason` is allowlisted. `truncation=reported` means the provider returned `length`; `not_reported` means a recognized other reason; otherwise it is `unknown`. None asserts that output was complete. The diagnostic contains no model/provider identifiers, prompts, documents or response text, stays in memory and clears when another translation starts.

Common token formats, credential assignments, URLs, local paths and email addresses are redacted before preview. Pattern-based redaction cannot identify arbitrary confidential prose: do not paste private documents or secrets. The native transport rejects SDK-added user data, breadcrumbs, traces, request URLs, extra fields and attachments. Only the reviewed description and fixed diagnostic fields are allowed.

## Sentry SDK and server contract

The pinned MIT `@sentry/browser` and `@sentry/core` 8.55.0 SDKs use an isolated BrowserClient/Scope, `captureFeedback`, optional `captureEvent`, and the official envelope serializer. The SDK is loaded only for an explicit send, with no global init/default integrations or automatic collection. A custom native Wails transport avoids browser CORS changes. It supports only this manual event/feedback subset, not Sentry tracing, native crashes, symbolication or arbitrary SDK payloads.

One general report sends a `feedback` item. An AI report first sends one fixed `event` exception, then a `feedback` item linked by `associated_event_id`. OS/architecture are appended to the feedback message from the reviewed diagnostic fields. Error values contain only category/finish reason/truncation. Each item goes to `POST /api/1/envelope/`; the bundled DSN contains only the existing public ingestion key, with no administrator credential. An explicit `SKILLVIEW_FEEDBACK_DSN` overrides it for deployment/testing; an empty value disables sending. HTTPS is required except loopback test endpoints. Redirects are rejected.

Only HTTP 200 with a bounded, valid `{accepted, duplicates, skipped}` receipt confirming exactly one accepted/duplicate item and zero skipped items counts as confirmed delivery. This backend commits SQLite before acknowledging. SDK flush confirms local serialization only, never delivery. The backend currently has one configured project; the DSN selects it. No behavior capture is enabled by this integration.

## Failure isolation

At most one report is in flight, with no waiting queue, at most two requests and a six-second total deadline. There are zero automatic retries. Closing the feedback window cancels its request and never blocks reading, translation or application exit. DNS/connection/TLS failures, timeout, 429, 5xx, redirects and malformed receipts only produce a feedback error. A 429 imposes an in-memory cooldown, normally 60 seconds. No raw server body or network details are displayed.

Manual retries keep the reviewed IDs and timestamp; the backend deduplicates identical normalized events. This also recovers a partially delivered AI report. A timeout has unknown delivery outcome. Closing the dialog discards its in-memory draft; editing/reviewing creates a new report. Reports are synthetic in automated acceptance tests; production-network acceptance is opt-in.
