# Claude Raw Traffic Audit Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Record and display the complete unredacted client and upstream lifecycle for Claude requests only.

**Architecture:** Keep client capture at the Gin middleware boundary and add an Anthropic-only wrapper around `GatewayService` upstream HTTP calls. Both emit exact-byte JSONL records keyed by the existing request ID; the admin API groups stages and the Vue admin page presents a readable four-stage audit view.

**Tech Stack:** Go, Gin, JSONL, Vue 3, TypeScript, Vitest, Tailwind CSS.

---

### Task 1: Shared Exact-Byte JSONL Writer

**Files:**
- Create: `backend/internal/audit/rawexchange/writer.go`
- Create: `backend/internal/audit/rawexchange/writer_test.go`
- Modify: `backend/internal/server/middleware/raw_exchange_logger.go`

- [ ] Write a failing test that appends concurrent records and recovers exact invalid UTF-8 through Base64.
- [ ] Run `go test ./internal/audit/rawexchange -count=1` and verify the package or API is missing.
- [ ] Implement one synchronized append writer, path resolution, exact-byte fields, and restrictive directory/file modes.
- [ ] Move middleware persistence to the shared writer without changing inbound capture behavior.
- [ ] Run package tests and the existing middleware raw-exchange tests.

### Task 2: Claude Upstream Attempt Capture

**Files:**
- Create: `backend/internal/service/claude_raw_exchange.go`
- Create: `backend/internal/service/claude_raw_exchange_test.go`
- Modify: `backend/internal/service/gateway_forward.go`
- Modify: `backend/internal/service/gateway_anthropic_passthrough.go`
- Modify: `backend/internal/service/gateway_count_tokens.go`
- Modify: `backend/internal/service/gateway_forward_as_responses.go`
- Modify: `backend/internal/service/gateway_forward_as_chat_completions.go`

- [ ] Write failing tests proving an Anthropic account records exact outbound request headers/body and inbound response headers/body under the context request ID.
- [ ] Add failing streaming tests proving reads are tee-captured without eager draining or altered bytes.
- [ ] Add failing tests for transport errors, retries/attempt numbers, and non-Anthropic accounts producing no record.
- [ ] Implement a `GatewayService` upstream-call wrapper that snapshots the request body, invokes `HTTPUpstream`, and wraps response bodies to finalize capture on EOF/Close.
- [ ] Replace GatewayService Claude-capable call sites, passing a stable operation and attempt number.
- [ ] Run focused GatewayService tests, then `go test ./internal/service -count=1`.

### Task 3: Correlated Admin API

**Files:**
- Modify: `backend/internal/handler/admin/ops_raw_exchange_log_handler.go`
- Modify: `backend/internal/handler/admin/ops_raw_exchange_log_handler_test.go`

- [ ] Write failing tests for grouping `client_exchange` and multiple `upstream_exchange` attempts by request ID.
- [ ] Write failing tests for filtering by user ID, client IP, model, status, request ID, body text, and stage.
- [ ] Implement newest-first grouped summaries while preserving every raw JSONL record in the response.
- [ ] Verify malformed or partially written trailing lines do not hide valid history.
- [ ] Run focused admin handler tests.

### Task 4: Readable Admin Viewer

**Files:**
- Modify: `frontend/src/api/admin/ops.ts`
- Modify: `frontend/src/api/__tests__/admin.ops.raw-exchange.spec.ts`
- Modify: `frontend/src/views/admin/RawExchangeLogsView.vue`
- Create: `frontend/src/views/admin/__tests__/RawExchangeLogsView.spec.ts`

- [ ] Write failing component tests for summary scanning, four-stage navigation, full body/header display, search, wrapping, and copy actions.
- [ ] Update API types for grouped exchanges and attempts.
- [ ] Refactor the page into a dense request table plus a detail workspace using existing admin components and semantic colors.
- [ ] Add accessible tabs/segmented controls for client request, upstream request, upstream response, and client response; retries remain individually selectable.
- [ ] Preserve full unredacted values, JSON formatting, raw/SSE modes, keyboard focus, responsive stacking, and stable dimensions.
- [ ] Run focused Vitest tests and `pnpm build`.

### Task 5: Configuration And Regression Verification

**Files:**
- Modify: `backend/internal/config/config.go`
- Modify: `backend/internal/config/config_test.go`
- Modify: `deploy/config.example.yaml`

- [ ] Write failing configuration tests for unlimited capture and the shared JSONL path.
- [ ] Remove body-size truncation from the myvps2 audit path while keeping the feature disabled by default outside configured environments.
- [ ] Run `go test ./internal/config ./internal/server/middleware ./internal/handler/admin ./internal/service -count=1`.
- [ ] Run frontend unit tests, type checking, and production build.
- [ ] Review the final diff to confirm only Claude traffic is captured and no redaction/truncation remains.
- [ ] Commit the implementation on `test/myvps2`; do not deploy or touch `goodserver` without a separate user instruction.

