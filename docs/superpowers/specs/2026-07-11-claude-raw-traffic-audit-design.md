# Claude Raw Traffic Audit Design

## Goal

Provide a test-environment audit trail that preserves the complete, unredacted Claude request lifecycle so administrators can investigate harmful or abusive user activity.

## Scope

Only Claude traffic is recorded. Administrative APIs, health checks, static files, and traffic routed to OpenAI, Gemini, Antigravity, or other platforms are excluded.

Each logical request records four stages under one request ID:

1. The original client request received by sub2api.
2. The exact HTTP request sub2api sends to the Claude/Anthropic upstream.
3. The exact HTTP response sub2api receives from the upstream, including SSE bytes.
4. The final HTTP response sub2api sends to the client.

Headers, URLs, query strings, bodies, binary bytes, credentials, and error details are stored without redaction or truncation. This feature is intended only for the isolated `myvps2` test environment.

## Architecture

The existing inbound middleware remains responsible for capturing the client-facing request and response. Claude gateway forwarding paths add an upstream capture boundary immediately around the actual HTTP call. Both boundaries emit records through one JSONL writer and use the request ID already carried in the request context.

Upstream capture is enabled only after the selected account/platform is known to be Anthropic. This avoids broad interception in the shared HTTP client and prevents unrelated platform traffic from entering the audit file.

Streaming responses are copied while being consumed rather than eagerly drained. The copied bytes are finalized when the stream reaches EOF or is closed, preserving the exact SSE stream without changing client delivery behavior.

## Record Format

Each JSONL entry contains a stage (`client_exchange` or `upstream_exchange`), request ID, timestamps, latency, account and user identifiers when available, model, platform, method, URL, headers, status, and body fields. Raw bytes are represented both as text and Base64 so invalid UTF-8 and binary content remain recoverable.

Failures before an upstream response are also recorded with the original request and transport error. Retries produce separate upstream attempt records with an attempt number, all tied to the same request ID.

## Viewer

The existing admin raw-log route is retained and restyled to match the current admin console. The primary view is a dense, readable request list with time, user, IP, model, status, latency, and request ID. Selecting a request opens a detail view organized into four clearly labeled stages.

Large JSON and SSE values use monospace text, wrapping controls, search, copy actions, and collapsible sections. Headers and bodies remain fully visible. The UI does not introduce a marketing layout, decorative cards, or a separate visual language.

The API groups related JSONL records by request ID while still allowing the administrator to inspect each original record and each retry attempt.

## Storage And Security

The JSONL file remains under `DATA_DIR/raw-exchange/raw-exchange.jsonl` unless explicitly configured otherwise. Appends are serialized and file permissions are restricted. Existing history remains readable after upgrades.

Because records intentionally contain API keys, authorization headers, prompts, and responses, access remains limited to authenticated administrators. No raw-log route is exposed to normal users.

## Main Synchronization

`origin/main` is merged into `test/myvps2` locally before implementation. Existing test-environment logging changes are preserved while conflicts are resolved in favor of current main architecture and build conventions. `main` itself is not modified.

## Testing

Tests first verify that Claude requests create correlated client and upstream records, exact bytes survive JSONL round trips, streaming responses are captured without changing delivery, retries remain separate, and non-Claude traffic produces no records. Admin API and frontend tests verify grouping, filtering, full-content display, and readable interaction states.

