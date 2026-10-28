# Feedback (local preparation)

The feedback dialog sends nothing until the user reviews a payload and chooses **Send reviewed report**. It reads no documents, translation responses, logs or credentials. Reports contain the entered description, app version, OS and architecture. The translation error row has a **Report this issue** action. An AI response format failure is kept only in memory and can be attached with code `SV_AI_RESPONSE_FORMAT`. Fixed categories distinguish `invalid_format`, `no_choices` and `incomplete_translations`; `finish_reason` is allowlisted. `truncation` is `reported` only for `length`, `not_reported` for a recognized other reason, and `unknown` when missing or unrecognized. This does not assert that an output is complete. Diagnostics never include model names, provider URLs, prompt text, documents or response content. The last diagnostic is cleared when another translation starts and is not persisted. General feedback remains available without diagnostics.

Common token formats, credential assignments, URLs, local paths and email addresses are redacted locally before preview. Pattern-based redaction cannot identify arbitrary confidential prose. The user must not paste private documents or secrets. There is no screenshot, session replay, automatic telemetry or background retry.

The service is not deployed. Both `SKILLVIEW_FEEDBACK_ENDPOINT` (origin only) and `SKILLVIEW_FEEDBACK_PROJECT_KEY` must be configured before sending is enabled. These are deployment settings; the project key must be public and ingestion-only. Never use an administrator token. HTTPS is required except for explicit loopback development endpoints. Redirects are rejected.

## Proposed v1 transport

`POST /api/v1/ingest`, JSON, header `X-Project-Key`.

Fields: `schema_version: 1`, `project_id: "skillview"`, `event_id` (32 lowercase hexadecimal characters), `kind` (`feedback` or `error`), `name`, `timestamp` (RFC3339), `release`, `message`, optional `error_code`, and `diagnostics` (`os`, `arch`, optional `ai_response` containing `category`, `finish_reason` and `truncation`).

Success: HTTP 201 for a stored report or HTTP 200 for an identical retry, with `{ "event_id": "...", "duplicate": false }` (true on retry). The service must atomically deduplicate `(project_id, event_id)` and reject the same ID with a different payload with HTTP 409. It must enforce that the project key belongs to `project_id`, limit payload sizes, rate-limit writes, and require separate administrator authentication for every query and state change.

The client keeps the reviewed payload and its ID for manual retries while the dialog remains open. A timeout is an unknown delivery outcome, not proof of failure. Closing the dialog discards its in-memory draft. Editing or reviewing again creates a new report. A confirmed receipt is shown only when the response contains the expected event ID. No server response body is displayed in an error message.

This is a small application-specific HTTP client, not a Sentry or OpenTelemetry SDK. The server's OpenAPI contract will become the shared protocol source after its project location is approved. No five-language SDK framework is planned.
